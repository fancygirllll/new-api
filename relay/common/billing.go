package common

import "github.com/gin-gonic/gin"

// BillingSettler abstracts the lifecycle operations of a billing session.
// It is implemented by service.BillingSession and stored on RelayInfo to avoid import cycles.
type BillingSettler interface {
	// Settle settles based on the actual quota consumed, computing
	// delta = actualQuota - preConsumedQuota, and adjusts both the funding source
	// (wallet/subscription) and the token quota.
	Settle(actualQuota int) error

	// Refund refunds all pre-consumed quota (funding source + token) and is idempotent.
	// It runs asynchronously through gopool. If already settled or refunded, it does nothing.
	Refund(c *gin.Context)

	// NeedsRefund reports whether the session still has a pre-consumed state that needs
	// refunding (neither settled nor refunded).
	NeedsRefund() bool

	// GetPreConsumedQuota returns the actual pre-consumed quota (may be 0 for trusted users).
	GetPreConsumedQuota() int

	// Reserve tops the pre-consumed quota up to the target value; if the target is not
	// higher than the current pre-consumed quota, it does nothing.
	Reserve(targetQuota int) error
}
