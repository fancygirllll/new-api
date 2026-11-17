package controller

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// distributorCustomerItem is a B4 customer list item (contract §4.2 B4).
type distributorCustomerItem struct {
	Id                   int    `json:"id"`
	Username             string `json:"username"`
	DisplayName          string `json:"display_name"`
	CreatedAt            int64  `json:"created_at"`
	TotalTopupCents      int64  `json:"total_topup_cents"`
	TotalCommissionCents int64  `json:"total_commission_cents"`
}

// listDistributorCustomerItems paginates the customers under a distributor and aggregates the two amount columns
// (shared parameterized implementation for B4/B7, extracted per plan 6.1 G-1; the distributorId source is decided by the caller:
// B4 narrows from the session, B7 takes path :id on the admin side; this function has no permission semantics).
// total_topup_cents is aggregated from TopUp.Money (SUM*100, F4 commission base semantics).
// total_commission_cents is wired from the aggregated commission flow amounts (contract appendix D5.3; positives and negatives offset, reversals are negative),
// the WHERE distributor_id = ? AND customer_id IN ? double predicate prevents cross-distributor reads (T-04-02-03).
func listDistributorCustomerItems(distributorId int, pageInfo *common.PageInfo) ([]distributorCustomerItem, int, error) {
	var users []model.User
	var total int64
	if err := model.DB.Model(&model.User{}).
		Where("inviter_id = ?", distributorId).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := model.DB.Where("inviter_id = ?", distributorId).
		Order("id desc").
		Limit(pageInfo.GetPageSize()).
		Offset(pageInfo.GetStartIdx()).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}

	// Commission aggregation: aggregate commission amounts for this page's customers in bulk; the flows amount column is already int64 cents, so reading it directly has no precision issues.
	customerIds := make([]int, 0, len(users))
	for _, u := range users {
		customerIds = append(customerIds, u.Id)
	}
	commissionByCustomer := make(map[int]int64, len(users))
	if len(customerIds) > 0 {
		var rows []struct {
			CustomerId int
			Total      int64
		}
		model.DB.Model(&model.CommissionFlow{}).
			Select("customer_id, COALESCE(SUM(amount_cents),0) AS total").
			Where("distributor_id = ? AND customer_id IN ?", distributorId, customerIds).
			Group("customer_id").
			Scan(&rows)
		for _, r := range rows {
			commissionByCustomer[r.CustomerId] = r.Total
		}
	}

	// Aggregate total_topup_cents per customer (SUM(money)*100, F4 commission base = amount actually paid)
	items := make([]distributorCustomerItem, 0, len(users))
	for _, u := range users {
		var totalMoney float64
		model.DB.Model(&model.TopUp{}).
			Where("user_id = ? AND status = ?", u.Id, common.TopUpStatusSuccess).
			Select("COALESCE(SUM(money), 0)").
			Scan(&totalMoney)
		items = append(items, distributorCustomerItem{
			Id:                   u.Id,
			Username:             u.Username,
			DisplayName:          u.DisplayName,
			CreatedAt:            u.CreatedAt,
			TotalTopupCents:      int64(math.Round(totalMoney * 100)),
			TotalCommissionCents: commissionByCustomer[u.Id], // Aggregated commission amount read directly; 0 when there are no flows
		})
	}
	return items, int(total), nil
}

// GetDistributorCustomers B4 distributor customer list (DistributorAuth).
// Server-side WHERE inviter_id = current distributor (ROLE-02, not relying on frontend isolation).
func GetDistributorCustomers(c *gin.Context) {
	distributorId := c.GetInt("id")
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

// GetDistributorCustomerTopups B5 top-up detail for one customer (DistributorAuth).
// IDOR protection: the server verifies the target customer's inviter_id == current distributor, otherwise 403 (contract §4.2 B5).
func GetDistributorCustomerTopups(c *gin.Context) {
	distributorId := c.GetInt("id")
	customerId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid customer id",
		})
		return
	}

	// IDOR protection: verify the target customer belongs to the current distributor
	var customer model.User
	if err := model.DB.Select("id, inviter_id").First(&customer, "id = ?", customerId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "customer not found",
		})
		return
	}
	if customer.InviterId != distributorId {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "access denied: customer does not belong to this distributor",
		})
		return
	}

	pageInfo := common.GetPageQuery(c)
	var topups []*model.TopUp
	var total int64
	model.DB.Model(&model.TopUp{}).
		Where("user_id = ?", customerId).
		Count(&total)
	if err := model.DB.Where("user_id = ?", customerId).
		Order("id desc").
		Limit(pageInfo.GetPageSize()).
		Offset(pageInfo.GetStartIdx()).
		Find(&topups).Error; err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(topups)
	common.ApiSuccess(c, pageInfo)
}

// GetDistributorProfile B6 the distributor's own profile (DistributorAuth).
// Returns aff_code / aff_link / customer_count.
func GetDistributorProfile(c *gin.Context) {
	distributorId := c.GetInt("id")
	user, err := model.GetUserById(distributorId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var customerCount int64
	model.DB.Model(&model.User{}).
		Where("inviter_id = ?", distributorId).
		Count(&customerCount)

	affLink := fmt.Sprintf("https://%s/register?aff=%s", c.Request.Host, user.AffCode)
	common.ApiSuccess(c, gin.H{
		"aff_code":       user.AffCode,
		"aff_link":       affLink,
		"customer_count": customerCount,
	})
}
