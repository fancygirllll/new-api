package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Distributor self statement surface (frozen shape per contract 01-CONTRACT.md §4.3 A10/A11, STMT-04/CUST-05).
// Permission middleware (exact role==5 match) is mounted at route registration; this file contains no permission wording (03-03 grep discipline).
// Server-side data narrowing: distributor_id is always taken from the session identity; never trust client parameters (ROLE-02).
// Query functions are reused as delivered in 04-02 (model.ListStatements / GetStatementWithAdjustments / ListStatementItems).

// GetSelfCommissionStatements A10 distributor statement list (own only, STMT-04).
// GET /api/commission/statements/self?p=&page_size=
func GetSelfCommissionStatements(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	// Server-side narrowing (ROLE-02): distributor_id is forced to the logged-in distributor
	items, total, err := model.ListStatements(c.GetInt("id"), "", "", pageInfo.GetPage(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

// GetSelfCommissionStatementItems A11 distributor statement items (ownership check, 403 on unauthorized access; items carry a redundant customer username, same shape as A6).
// GET /api/commission/statements/self/:id/items?p=&page_size=
func GetSelfCommissionStatementItems(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "Invalid statement ID")
		return
	}
	stmt, _, err := model.GetStatementWithAdjustments(int64(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "Statement not found")
			return
		}
		common.ApiError(c, err)
		return
	}
	if stmt.DistributorId != c.GetInt("id") { // Ownership check (Pitfall 3: 403 on unauthorized access)
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "You are not allowed to access this statement"})
		return
	}
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListStatementItems(int64(id), pageInfo.GetPage(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}
