package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// VoidTopUp A13 void a top-up order (super-admin middleware mounted at the route layer, frozen per contract 01-CONTRACT.md appendix C1).
// POST /api/topup/:tradeNo/void  request body {"reason": string} (required, for the audit trail)
//
// Response envelope (project convention: /api/* always HTTP 200 + {success, message, data}; do not hand-write 4xx):
//   - success -> {"success": true}
//   - reason missing/blank -> success:false (400 semantics)
//   - ErrTopUpNotFound / ErrTopUpStatusInvalid / ErrAlreadyStatemented → via the existing sentinel
//     returned by common.ApiError (contract C1: voiding an invoiced top-up rolls back entirely and reports an error; partial voiding is forbidden)
//
// Transaction semantics (model.VoidTopUp runs in a single all-or-nothing transaction): 1) Status -> refunded 2) deduct the quota
// Amount×QuotaPerUnit (negative balance allowed) 3) create a negative reversal flow entry 4) write the audit trail.
func VoidTopUp(c *gin.Context) {
	tradeNo := c.Param("tradeNo")
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		common.ApiErrorMsg(c, "Void reason cannot be empty")
		return
	}
	operatorId := c.GetInt("id")
	if err := model.VoidTopUp(tradeNo, req.Reason, operatorId); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
