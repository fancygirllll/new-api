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

// GetCommissionStatements A4 admin statement list (AdminAuth, frozen shape per contract 01-CONTRACT.md §4.3 A4).
// GET /api/commission/statements?distributor_id=&period=&status=&p=&page_size=
// Three optional filters: reject invalid distributor_id (never trust client input); skip the predicate when period/status is empty.
func GetCommissionStatements(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	distributorId := 0
	if s := c.Query("distributor_id"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			common.ApiErrorMsg(c, "Invalid distributor ID")
			return
		}
		distributorId = v
	}
	items, total, err := model.ListStatements(distributorId, c.Query("period"), c.Query("status"), pageInfo.GetPage(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

// GetCommissionStatement A5 statement detail (AdminAuth, contract §4.3 A5: statement body + adjustments array).
// GET /api/commission/statements/:id
// Validation chain: id parse failure -> HTTP 400 (precedent in commission_rate.go:41-48) -> not found -> success=false.
func GetCommissionStatement(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid statement ID",
		})
		return
	}
	stmt, adjustments, err := model.GetStatementWithAdjustments(int64(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "Statement not found")
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"statement":   stmt,
		"adjustments": adjustments,
	})
}

// GetCommissionStatementItems A6 paginated statement items (AdminAuth, contract §4.3 A6: items carry a redundant customer username).
// GET /api/commission/statements/:id/items?p=&page_size=
func GetCommissionStatementItems(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid statement ID",
		})
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

// GetCommissionStatementSummary A14 cross-distributor summary report (AdminAuth, frozen shape per contract appendix D3).
// GET /api/commission/statements/summary?period=YYYY-MM
// → data: {period, items[{distributor_id,username,total_commission_cents,adjusted_cents,statement_status,statement_id}], totals{total_commission_cents,adjusted_cents,distributor_count}}
// period is required; no statements for the period -> empty items array + zero totals (200, satisfies ROADMAP acceptance item 4).
func GetCommissionStatementSummary(c *gin.Context) {
	period := c.Query("period")
	if period == "" {
		common.ApiErrorMsg(c, "period is required")
		return
	}
	summary, err := model.SummarizeStatements(period)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}
