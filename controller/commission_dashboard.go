package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// Distributor commission dashboard (contract 01-CONTRACT.md §4.3 A9/A12 frozen shape + v1.5 appendix F, CUST-03/STMT-04/CUST-05).
// Permission middleware (exact role==5 match) is mounted at route registration; this file contains no permission wording (03-03 grep discipline).
// Server-side data narrowing: distributor_id is always taken from the session identity; never trust client parameters (ROLE-02).
// A9/A12 take no request parameters and have no pagination: the handler only derives the session identity -> model aggregation -> envelope (zero business logic,
// aggregation rules and the four-card identities all live in model/commission_dashboard.go).

// GetCommissionDashboard A9 distributor commission dashboard (four summary items + appendix F2 counts; server-side aggregation is the single source of truth).
// GET /api/commission/dashboard
func GetCommissionDashboard(c *gin.Context) {
	distributorId := c.GetInt("id") // derived from the server-side session; never trust client parameters (ROLE-02)
	data, err := model.GetCommissionDashboard(distributorId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

// GetCurrentCommission A12 real-time preview of un-invoiced commission (period/count/amount as three frozen fields, bound to the same predicate as A9-1).
// GET /api/commission/current
func GetCurrentCommission(c *gin.Context) {
	distributorId := c.GetInt("id") // derived from the server-side session; never trust client parameters (ROLE-02)
	data, err := model.GetCurrentPending(distributorId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}
