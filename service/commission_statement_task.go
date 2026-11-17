package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"github.com/bytedance/gopkg/util/gopool"
)

// Monthly statement generation task (contract 01-CONTRACT.md appendix D2 frozen: mirrors the subscription_reset_task pattern).
//
// Scheduling semantics: tick every minute; `day >= clamp(payoutDay)` is permissive - any time on or after the payout day within the month
// (process restart, downtime catch-up, mid-month config change) automatically catches up the previous period's statement; no "already ran today" marker needed;
// Run once at startup to cover the case where the process starts after the payout day.
//
// Three layers of deduplication (contract §5/D2 frozen; do not switch to a cron library, distributed lock, or system_task DB lease):
//  1. IsMasterNode gate (blocks slave nodes; multiple masters may coexist legally, so this layer alone is not enough)
//  2. uk_stmt_period(distributor_id, period) unique index (final DB-level defense)
//  3. FOR UPDATE on the distributor's users row (serialized within the transaction, model layer)
//
// Gating discipline (appendix D5.1): statement generation is not gated by the commission master switch - correction and reconciliation are ledger-integrity operations;
// the switch only controls new bookkeeping entry points.
//
// Time zone discipline (contract C3 + Pitfall 8): always use now's own Location (the server's local time zone;
// deployment assumes TZ=Asia/Shanghai, already set in docker-compose); do not mix in time.UTC.

var (
	statementTaskOnce    sync.Once
	statementTaskRunning atomic.Bool
)

// statementTaskTickInterval is the statement generation tick period (mode A: one day-of-month check per minute).
const statementTaskTickInterval = time.Minute

// StartCommissionStatementTask starts the monthly statement generation task (master node only).
// Registration point: after StartSubscriptionQuotaResetTask() in main.go.
func StartCommissionStatementTask() {
	statementTaskOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("commission statement task started: node=%s tick=%s", common.NodeName, statementTaskTickInterval))
			// Run once at startup: covers catch-up when the process starts after the payout day (D2 permissive semantics)
			runStatementGenerationOnce(time.Now())
			ticker := time.NewTicker(statementTaskTickInterval)
			defer ticker.Stop()
			for now := range ticker.C {
				runStatementGenerationOnce(now)
			}
		})
	})
}

// payoutDayClamped clamps for short months (contract §3.4: use the last day when the month has fewer days).
// Example: payoutDay=31 in February (28/29 days) returns 28/29 - otherwise February would be skipped forever (Pitfall 2);
// Extracted as a pure function for unit testing (no DB/clock access).
func payoutDayClamped(now time.Time) int {
	lastDay := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()
	if common.CommissionPayoutDay > lastDay {
		return lastDay
	}
	return common.CommissionPayoutDay
}

// runStatementGenerationOnce performs a single statement generation check (now is parameterized so tests can supply any date;
// this is the test seam for Pitfalls 1/2/8). CAS prevents re-entry: only one executor takes effect for concurrent/overlapping ticks.
func runStatementGenerationOnce(now time.Time) {
	if !statementTaskRunning.CompareAndSwap(false, true) {
		return
	}
	defer statementTaskRunning.Store(false)

	// Payout day not reached (permissive semantics: >= attempts; idempotency is guaranteed by the three model-layer dedup layers, so repeated runs are naturally no-ops)
	if now.Day() < payoutDayClamped(now) {
		return
	}

	n, err := model.GenerateDueStatements(now)
	if err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("commission statement generation failed: node=%s err=%v", common.NodeName, err))
		return
	}
	if n > 0 {
		logger.LogInfo(context.Background(), fmt.Sprintf("commission statements generated: node=%s period_count=%d", common.NodeName, n))
	}
}
