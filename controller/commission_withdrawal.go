package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// Per-withdrawal WDRAW surface (frozen shape per contract 01-CONTRACT.md appendix E2 A15-A21, v1.4).
// Permission middleware is mounted at route registration (distributor side exact role==5 / admin side threshold role>=10),
// this file contains no permission wording (03-03 grep discipline); no endpoint branches on the master commission switch (D-14/Q5=A).
//
// The controller makes no business state decisions - the five-state transition allowlist and statement linkage all live inside the model Tx;
// this file only does: strconv binding -> JSON binding -> required-field pre-checks -> Tx -> envelope.
//
// Audit discipline (A7/A8 precedent): A18-A21 write the trail after the model Tx returns nil (transaction committed)
// admin operation audit trails run outside the transaction - no success trail on failure paths; A15 has no out-of-transaction audit (the application trail
// is the withdrawal row itself, and the distributor side has no fallback audit chain); admin write operations automatically get the middleware fallback
// audit (the third of three layers; the double entries for A18-A21 reflect existing semantics).
// Error paths always use the envelope (success=false) - the fallback audit's success determination depends on it (Pitfall 4/T-05-01-10).

// CreateWithdrawal A15 starts a withdrawal request (full statement amount, D-08: no amount input on the request).
// POST /api/commission/withdrawals  {"statement_id": int64}
func CreateWithdrawal(c *gin.Context) {
	var req struct {
		StatementId int64 `json:"statement_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "Invalid request parameter")
		return
	}
	if _, err := model.CreateWithdrawalTx(req.StatementId, c.GetInt("id")); err != nil {
		if errors.Is(err, model.ErrWithdrawalNotOwner) { // ownership violation -> 403 (CUST-05/ROLE-02)
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": err.Error()})
			return
		}
		common.ApiError(c, err) // business errors such as the payable allowlist go through the envelope (Pitfall 6 message passthrough)
		return
	}
	common.ApiSuccess(c, nil) // Appendix E2 frozen response shape (A7/A8 precedent)
}

// GetSelfWithdrawals A16 distributor withdrawal list (server-side WHERE distributor_id = self).
// GET /api/commission/withdrawals/self?p=&page_size=
func GetSelfWithdrawals(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListWithdrawals(c.GetInt("id"), 0, "", pageInfo.GetPage(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

// GetWithdrawals A17 admin withdrawal list (distributor_id/status filters optional; invalid distributor_id rejected).
// GET /api/commission/withdrawals?distributor_id=&status=&p=
func GetWithdrawals(c *gin.Context) {
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
	items, total, err := model.ListWithdrawals(0, distributorId, c.Query("status"), pageInfo.GetPage(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

// withdrawalAmountCents reads the statement settle amount outside the transaction as an audit snapshot (D-08 live-value fallback;
// used only for audit log content, never in any write path).
func withdrawalAmountCents(statementId int64) int64 {
	if stmt, _, err := model.GetStatementWithAdjustments(statementId); err == nil {
		return stmt.SettleAmountCents
	}
	return 0
}

// AcceptWithdrawal A18 accept (pending->reviewing, no statement changes).
// POST /api/commission/withdrawals/:id/accept (no body)
func AcceptWithdrawal(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "Invalid withdrawal ID")
		return
	}
	wd, err := model.AcceptWithdrawalTx(int64(id), c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// Audit trail after the transaction commits (no success trail on failure paths)
	model.RecordLog(c.GetInt("id"), model.LogTypeManage, fmt.Sprintf("Withdrawal accepted withdrawal_no=%s statement_id=%d", wd.WithdrawalNo, wd.StatementId))
	common.ApiSuccess(c, nil)
}

// RejectWithdrawal A19 reject (only from reviewing, Q2=B; the statement reverts to payable in the same transaction).
// POST /api/commission/withdrawals/:id/reject  {"reason": string required}
func RejectWithdrawal(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "Invalid withdrawal ID")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "Invalid request parameter")
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		common.ApiErrorMsg(c, "Rejection reason cannot be empty")
		return
	}
	operatorId := c.GetInt("id")
	wd, err := model.RejectWithdrawalTx(int64(id), req.Reason, operatorId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// Audit trail after the transaction commits: embed the withdrawal_no/statement_id/amount_cents snapshot/reason
	// (amount is read live from statement.settle_amount_cents, D-08 audit fallback)
	model.RecordLog(operatorId, model.LogTypeManage, fmt.Sprintf("Withdrawal rejected withdrawal_no=%s statement_id=%d amount_cents=%d reason=%s", wd.WithdrawalNo, wd.StatementId, withdrawalAmountCents(wd.StatementId), req.Reason))
	common.ApiSuccess(c, nil)
}

// ApproveWithdrawal A20 approve (reviewing->approved, the statement stays withdrawing, Q1=B).
// POST /api/commission/withdrawals/:id/approve (no body)
func ApproveWithdrawal(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "Invalid withdrawal ID")
		return
	}
	wd, err := model.ApproveWithdrawalTx(int64(id), c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// Audit trail after the transaction commits (no success trail on failure paths)
	model.RecordLog(c.GetInt("id"), model.LogTypeManage, fmt.Sprintf("Withdrawal approved withdrawal_no=%s statement_id=%d", wd.WithdrawalNo, wd.StatementId))
	common.ApiSuccess(c, nil)
}

// MarkWithdrawalPaid A21 record payout (only from approved, Q1=B; the statement becomes settled in the same transaction).
// POST /api/commission/withdrawals/:id/paid  {"voucher_no": string required}
func MarkWithdrawalPaid(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "Invalid withdrawal ID")
		return
	}
	var req struct {
		VoucherNo string `json:"voucher_no"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "Invalid request parameter")
		return
	}
	if strings.TrimSpace(req.VoucherNo) == "" {
		common.ApiErrorMsg(c, "Payout voucher number cannot be empty")
		return
	}
	operatorId := c.GetInt("id")
	wd, err := model.MarkWithdrawalPaidTx(int64(id), req.VoucherNo, operatorId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// Audit trail after the transaction commits: embed the withdrawal_no/voucher_no/amount_cents snapshot
	model.RecordLog(operatorId, model.LogTypeManage, fmt.Sprintf("Withdrawal paid withdrawal_no=%s statement_id=%d amount_cents=%d voucher_no=%s", wd.WithdrawalNo, wd.StatementId, withdrawalAmountCents(wd.StatementId), req.VoucherNo))
	common.ApiSuccess(c, nil)
}
