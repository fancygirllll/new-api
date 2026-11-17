package model

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// Attribution management error variables (contract 01-CONTRACT.md §4.2 B1/B2 + §5 idempotency and concurrency conventions).
// The controller layer maps sentinel errors to HTTP status codes:
//   - ErrInvalidAffCode / ErrNotDistributor / ErrAlreadyBound → 400
//   - ErrWindowExpired → 403
//   - ErrUserNotFound / ErrDistributorNotFound → 404
var (
	ErrInvalidAffCode      = errors.New("aff code not found")
	ErrNotDistributor      = errors.New("aff code owner is not a distributor")
	ErrAlreadyBound        = errors.New("user already has an inviter")
	ErrWindowExpired       = errors.New("rebind window expired")
	ErrUserNotFound        = errors.New("user not found")
	ErrDistributorNotFound = errors.New("distributor not found")
)

// CanRebind reports whether the customer is still within the self-service rebind window (contract §2.4 + §4.1 M2).
// Pure function: depends only on its arguments and time.Now(), with no DB or global mutable state access.
//   - mode=="unlimited" -> always true (only constrained by inviter_id=0 at any time)
//   - mode=="days" -> userCreatedAt + days*86400 > now (rebind allowed within the window)
//   - otherwise -> false (defensive; should not occur in normal configuration)
func CanRebind(userCreatedAt int64, mode string, days int) bool {
	switch mode {
	case "unlimited":
		return true
	case "days":
		if days <= 0 {
			return false
		}
		return userCreatedAt+int64(days)*86400 > time.Now().Unix()
	default:
		return false
	}
}

// SelfBindAttribution performs a self-service rebind inside a transaction (contract §4.2 B1 + §5 idempotency and concurrency conventions).
// Two deduplication layers:
//  1. Pre-check InviterId != 0 -> ErrAlreadyBound (fast rejection, reducing contention on the conditional UPDATE)
//  2. Conditional UPDATE inviter_id=0 -> affected=0 means ErrAlreadyBound (final defense against concurrent duplication)
//
// Window validation runs first (days mode: created_at + days*86400 > now; unlimited always passes).
// Invitation code not owned by a distributor -> ErrNotDistributor (prevents binding to a regular inviter posing as a distributor).
// On success, write AttributionChange{source: "self_bind"} in the same transaction.
//
// Note: this path is physically isolated from AdminBindAttribution - the self-service side uses a conditional UPDATE (deduplication),
// while the admin side uses an unconditional UPDATE (F5 may overwrite existing attribution).
func SelfBindAttribution(tx *gorm.DB, userId int, affCode string) error {
	if affCode == "" {
		return ErrInvalidAffCode
	}
	// 1. User corresponding to the invitation code
	inviterId, err := GetUserIdByAffCode(affCode)
	if err != nil || inviterId == 0 {
		return ErrInvalidAffCode
	}
	// 2. Verify the inviter is a distributor (prevents binding to a regular inviter)
	var inviter User
	if e := tx.Select("id, role").First(&inviter, "id = ?", inviterId).Error; e != nil {
		return ErrInvalidAffCode
	}
	if inviter.Role != common.RoleDistributorUser {
		return ErrNotDistributor
	}
	// 3. Load the current user (to get CreatedAt + InviterId)
	var user User
	if e := tx.First(&user, "id = ?", userId).Error; e != nil {
		return ErrUserNotFound
	}
	// 4. First dedup layer: pre-check existing attribution
	if user.InviterId != 0 {
		return ErrAlreadyBound
	}
	// 5. Window validation first
	if !CanRebind(user.CreatedAt, common.AffRebindWindowMode, common.AffRebindWindowDays) {
		return ErrWindowExpired
	}
	// 6. Conditional UPDATE (prevents concurrent duplicate binding; affected=0 -> already bound concurrently)
	res := tx.Model(&User{}).
		Where("id = ? AND inviter_id = 0", userId).
		Update("inviter_id", inviterId)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrAlreadyBound
	}
	// 7. Audit in the same transaction (source=self_bind, operator_id=userId)
	return tx.Create(&AttributionChange{
		UserId:       userId,
		OldInviterId: 0,
		NewInviterId: inviterId,
		Source:       "self_bind",
		OperatorId:   userId,
		Reason:       "self",
	}).Error
}

// AdminBindAttribution performs an admin rebind inside a transaction (contract §4.2 B2 + §5 F5 unconditional overwrite).
// Physically isolated from SelfBindAttribution: the self-service side uses conditional UPDATE inviter_id=0 (deduplication),
// while the admin side uses an unconditional UPDATE (F5, may overwrite existing attribution, auditing both Old/New).
// Validation: target user exists + distributor exists + distributor.Role==5.
func AdminBindAttribution(tx *gorm.DB, userId int, distributorId int, operatorId int, reason string) error {
	var user User
	if err := tx.First(&user, "id = ?", userId).Error; err != nil {
		return ErrUserNotFound
	}
	var distributor User
	if err := tx.First(&distributor, "id = ?", distributorId).Error; err != nil {
		return ErrDistributorNotFound
	}
	if distributor.Role != common.RoleDistributorUser {
		return ErrNotDistributor
	}
	oldInviterId := user.InviterId
	if err := tx.Model(&User{}).Where("id = ?", userId).Update("inviter_id", distributorId).Error; err != nil {
		return err
	}
	return tx.Create(&AttributionChange{
		UserId:       userId,
		OldInviterId: oldInviterId,
		NewInviterId: distributorId,
		Source:       "admin_bind",
		OperatorId:   operatorId,
		Reason:       reason,
	}).Error
}

// GetAttributionChanges paginates attribution change audit records (contract §4.2 B3).
func GetAttributionChanges(userId int, page *common.PageInfo) ([]AttributionChange, int64, error) {
	var changes []AttributionChange
	var total int64
	query := DB.Model(&AttributionChange{})
	if userId != 0 {
		query = query.Where("user_id = ?", userId)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id desc").Limit(page.GetPageSize()).Offset(page.GetStartIdx()).Find(&changes).Error; err != nil {
		return nil, 0, err
	}
	return changes, total, nil
}
