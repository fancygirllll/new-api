package middleware

import (
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	// SecureVerificationSessionKey is the session key for secure verification (kept in sync with the controller).
	SecureVerificationSessionKey       = "secure_verified_at"
	secureVerificationMethodSessionKey = "secure_verified_method"
	// SecureVerificationTimeout is the verification validity period in seconds.
	SecureVerificationTimeout = 300 // 5 minutes
)

// SecureVerificationRequired is the secure verification middleware.
// It checks whether the user passed secure verification within the validity period.
// If not verified or the verification has expired, it returns 401.
func SecureVerificationRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check whether the user is signed in.
		userId := c.GetInt("id")
		if userId == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Not signed in.",
			})
			c.Abort()
			return
		}

		// Check the verification timestamp in the session.
		session := sessions.Default(c)
		verifiedAtRaw := session.Get(SecureVerificationSessionKey)

		if verifiedAtRaw == nil {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Secure verification required.",
				"code":    "VERIFICATION_REQUIRED",
			})
			c.Abort()
			return
		}

		verifiedAt, ok := verifiedAtRaw.(int64)
		if !ok {
			// Invalid session data format.
			clearSecureVerificationSession(session)
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Verification state is invalid. Please verify again.",
				"code":    "VERIFICATION_INVALID",
			})
			c.Abort()
			return
		}

		// Check whether the verification has expired.
		elapsed := time.Now().Unix() - verifiedAt
		if elapsed >= SecureVerificationTimeout {
			// Verification expired; clear the session.
			clearSecureVerificationSession(session)
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Verification has expired. Please verify again.",
				"code":    "VERIFICATION_EXPIRED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func clearSecureVerificationSession(session sessions.Session) {
	session.Delete(SecureVerificationSessionKey)
	session.Delete(secureVerificationMethodSessionKey)
	_ = session.Save()
}

// OptionalSecureVerification is an optional secure verification middleware.
// If the user is verified, it sets a flag in the context but does not block the request.
// It is used in scenarios that need to distinguish whether the user is verified.
func OptionalSecureVerification() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.GetInt("id")
		if userId == 0 {
			c.Set("secure_verified", false)
			c.Next()
			return
		}

		session := sessions.Default(c)
		verifiedAtRaw := session.Get(SecureVerificationSessionKey)

		if verifiedAtRaw == nil {
			c.Set("secure_verified", false)
			c.Next()
			return
		}

		verifiedAt, ok := verifiedAtRaw.(int64)
		if !ok {
			c.Set("secure_verified", false)
			c.Next()
			return
		}

		elapsed := time.Now().Unix() - verifiedAt
		if elapsed >= SecureVerificationTimeout {
			clearSecureVerificationSession(session)
			c.Set("secure_verified", false)
			c.Next()
			return
		}

		c.Set("secure_verified", true)
		c.Set("secure_verified_at", verifiedAt)
		c.Next()
	}
}

// ClearSecureVerification clears the secure verification state.
// It is used when the user signs out or re-verification must be forced.
func ClearSecureVerification(c *gin.Context) {
	session := sessions.Default(c)
	clearSecureVerificationSession(session)
}
