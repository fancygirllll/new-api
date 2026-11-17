package controller

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// Super-admin adjustment surface (frozen shape per contract 01-CONTRACT.md §4.3 A7/A8, COMM-05/06).
// Permission middleware (super-admin threshold) is mounted at route registration; this file contains no permission wording (03-03 grep discipline).
// Request shape is frozen - do not add or remove fields:
//   A7 POST /api/commission/flows/manual  {"distributor_id":int,"flow_type":string,"amount_cents":int64,"reason":string}
//   A8 POST /api/commission/statements/:id/adjustments  {"delta_cents":int64,"reason":string} (only allowed for payable statements)
//
// Audit discipline (VoidTopUp precedent): write the audit trail after the model function returns nil (transaction committed)
// Execute outside the transaction - no success trail on failure paths; the trail uses the global DB internally, so calling it inside a transaction would bypass tx and it should not disappear on rollback.
// Controller side of the two-layer reason validation: TrimSpace required (missing -> 400 semantics envelope).

// CreateManualCommissionFlow A7 super-admin manually records/reverses a flow.
// Most of the validation chain lives in model.CreateManualFlow (flow_type allowlist, positive amount, exact target role check),
// the controller only binds the request and pre-checks that reason is required.
func CreateManualCommissionFlow(c *gin.Context) {
	var req struct {
		DistributorId int    `json:"distributor_id"`
		FlowType      string `json:"flow_type"`
		AmountCents   int64  `json:"amount_cents"`
		Reason        string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "Invalid request parameter")
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		common.ApiErrorMsg(c, "Reason cannot be empty")
		return
	}
	operatorId := c.GetInt("id")
	if err := model.CreateManualFlow(req.DistributorId, req.FlowType, req.AmountCents, req.Reason, operatorId); err != nil {
		common.ApiError(c, err)
		return
	}
	// Audit trail after the transaction commits (no success trail on failure paths)
	model.RecordLog(operatorId, model.LogTypeManage, fmt.Sprintf("Manual record/reversal distributor_id=%d flow_type=%s amount_cents=%d reason=%s", req.DistributorId, req.FlowType, req.AmountCents, req.Reason))
	common.ApiSuccess(c, nil)
}

// CreateStatementAdjustment A8 super-admin statement adjustment (only for payable statements; allowlist enforced in the model layer).
// The server recomputes the settle amount from the statement's current totals (never trusting the client); the original statement Total/items are untouched.
func CreateStatementAdjustment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "Invalid statement ID")
		return
	}
	var req struct {
		DeltaCents int64  `json:"delta_cents"`
		Reason     string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "Invalid request parameter")
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		common.ApiErrorMsg(c, "Reason cannot be empty")
		return
	}
	operatorId := c.GetInt("id")
	if err := model.CreateStatementAdjustmentTx(int64(id), req.DeltaCents, req.Reason, operatorId); err != nil {
		common.ApiError(c, err)
		return
	}
	// Audit trail after the transaction commits (no success trail on failure paths)
	model.RecordLog(operatorId, model.LogTypeManage, fmt.Sprintf("Statement adjustment statement_id=%d delta_cents=%d reason=%s", id, req.DeltaCents, req.Reason))
	common.ApiSuccess(c, nil)
}
