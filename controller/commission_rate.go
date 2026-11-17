package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetCommissionRates A1 rate list (AdminAuth, frozen shape per contract 01-CONTRACT.md §4.3 A1).
// GET /api/commission/rates?p=&page_size=
// items are CommissionRateDetail (CommissionRate embedded flat + redundant distributor username for display).
func GetCommissionRates(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListRates(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

// SetCommissionRate A2 set a rate (AdminAuth, contract §4.3 A2 + §3.2 same-transaction history frozen).
// PUT /api/commission/rates/:distributorId  request body {"rate_bp": 500}
//
// Validation chain (return on failure; T-03-02-02 never trust any client value):
//  1. :distributorId strconv.Atoi parse failure -> HTTP 400 (distributor.go precedent)
//  2. rate_bp out of [0,10000] -> 400 semantics (RATE-01: basis points enforced server-side)
//  3. The target user exists and Role == RoleDistributorUser (exact ==, AdminBindAttribution pattern,
//     no threshold checks - admin=10/root=100 are not distributors) -> 400 semantics
//  4. Write rate + history in the same transaction (rate first, then history, per RESEARCH anti-pattern);
//     operator is taken from the session id (T-03-02-03 audit trail)
func SetCommissionRate(c *gin.Context) {
	// 1. Parse path parameters (note the frozen parameter name is :distributorId, not :id)
	distributorId, err := strconv.Atoi(c.Param("distributorId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid distributor ID",
		})
		return
	}
	// 2. Bind the request body + validate the basis-point range (0~10000, enforced server-side)
	var req struct {
		RateBp int `json:"rate_bp"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request parameter",
		})
		return
	}
	if req.RateBp < 0 || req.RateBp > 10000 {
		common.ApiErrorMsg(c, "rate_bp must be between 0 and 10000 (basis points)")
		return
	}
	// 3. The target must be a distributor (exact == RoleDistributorUser)
	target, err := model.GetUserById(distributorId, false)
	if err != nil || target.Role != common.RoleDistributorUser {
		common.ApiErrorMsg(c, "Target user is not a distributor")
		return
	}
	// 4. Write rate + history in the same transaction (frozen per contract §3.2)
	operatorId := c.GetInt("id")
	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		return model.UpsertRateWithHistory(tx, distributorId, req.RateBp, operatorId)
	}); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

// GetCommissionRateHistory A3 rate change history (AdminAuth, contract §4.3 A3, RATE-03).
// GET /api/commission/rates/:distributorId/history?p=&page_size=
// items are CommissionRateHistory (operator/old/new/complete timestamp).
func GetCommissionRateHistory(c *gin.Context) {
	distributorId, err := strconv.Atoi(c.Param("distributorId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid distributor ID",
		})
		return
	}
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.GetRateHistories(distributorId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}
