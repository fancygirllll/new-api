package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// affRebindRequest is the self-service rebind request body (contract §4.2 B1).
type affRebindRequest struct {
	AffCode string `json:"aff_code"`
}

// AffRebind is the B1 self-service rebind handler (UserAuth).
// A conditional UPDATE inviter_id=0 inside the transaction prevents concurrent duplicate binding; the window check runs first.
// Error mapping (contract §4.2 B1 + plan 02-03 Task 1):
//   - empty aff_code -> 400 invalid parameter
//   - errInvalidAffCode / errNotDistributor → 400
//   - errAlreadyBound -> 400 (already attributed)
//   - errWindowExpired -> 403 (window expired)
//   - errUserNotFound → 404
func AffRebind(c *gin.Context) {
	var req affRebindRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.AffCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgInvalidParams),
		})
		return
	}
	userId := c.GetInt("id")
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		return model.SelfBindAttribution(tx, userId, req.AffCode)
	})
	if err != nil {
		switch {
		case errors.Is(err, model.ErrInvalidAffCode),
			errors.Is(err, model.ErrNotDistributor),
			errors.Is(err, model.ErrAlreadyBound):
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": err.Error(),
			})
		case errors.Is(err, model.ErrWindowExpired):
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": err.Error(),
			})
		case errors.Is(err, model.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": err.Error(),
			})
		default:
			common.ApiError(c, err)
		}
		return
	}
	common.ApiSuccess(c, gin.H{"success": true})
}

// adminBindRequest is the admin rebind request body (contract §4.2 B2).
type adminBindRequest struct {
	UserId        int    `json:"user_id"`
	DistributorId int    `json:"distributor_id"`
	Reason        string `json:"reason"`
}

// AdminBindAttribution is the B2 admin rebind handler (RootAuth).
// F5 unconditional overwrite: not constrained by inviter_id=0, can correct attribution for already-attributed customers.
// Write the audit AttributionChange{source: "admin_bind"} in the same transaction.
// reason is required (explicitly stated in contract §4.2 B2).
func AdminBindAttribution(c *gin.Context) {
	var req adminBindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgInvalidParams),
		})
		return
	}
	if req.UserId == 0 || req.DistributorId == 0 || req.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgInvalidParams),
		})
		return
	}
	operatorId := c.GetInt("id")
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		return model.AdminBindAttribution(tx, req.UserId, req.DistributorId, operatorId, req.Reason)
	})
	if err != nil {
		switch {
		case errors.Is(err, model.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": err.Error(),
			})
		case errors.Is(err, model.ErrDistributorNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": err.Error(),
			})
		case errors.Is(err, model.ErrNotDistributor):
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": err.Error(),
			})
		default:
			common.ApiError(c, err)
		}
		return
	}
	common.ApiSuccess(c, gin.H{"success": true})
}

// GetAttributionChanges is the B3 attribution audit query handler (AdminAuth).
// Paginated query of AttributionChange records, filterable by user_id.
func GetAttributionChanges(c *gin.Context) {
	userId := 0
	if uidStr := c.Query("user_id"); uidStr != "" {
		if uid, err := strconv.Atoi(uidStr); err == nil {
			userId = uid
		}
	}
	pageInfo := common.GetPageQuery(c)
	changes, total, err := model.GetAttributionChanges(userId, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(changes)
	common.ApiSuccess(c, pageInfo)
}

// GetDistributorCustomersForAdmin B7 lets admins view the customer list of any distributor (AdminAuth).
// contract appendix G (plan 6.1 G-1): the row shape is identical to B4 distributorCustomerItem,
// the aggregation query reuses the B4 implementation (listDistributorCustomerItems); :id comes from the path,
// non-numeric -> 400 (wording follows the B5 precedent in the same scenario, without depending on i18n init); permissions are guaranteed by route middleware,
// the controller has zero permission logic.
func GetDistributorCustomersForAdmin(c *gin.Context) {
	distributorId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid distributor id",
		})
		return
	}
	pageInfo := common.GetPageQuery(c)
	items, total, err := listDistributorCustomerItems(distributorId, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(total)
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}
