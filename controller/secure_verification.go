package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	// SecureVerificationSessionKey means the user has fully passed secure verification.
	SecureVerificationSessionKey       = "secure_verified_at"
	secureVerificationMethodSessionKey = "secure_verified_method"
	secureVerificationMethod2FA        = "2fa"
	secureVerificationMethodPasskey    = "passkey"
	// PasskeyReadySessionKey means WebAuthn finished and /api/verify can finalize step-up verification.
	PasskeyReadySessionKey = "secure_passkey_ready_at"
	// SecureVerificationTimeout is the verification validity period (seconds)
	SecureVerificationTimeout = 300 // 5 minutes
	// PasskeyReadyTimeout is the validity period of the passkey ready flag (seconds)
	PasskeyReadyTimeout = 60
)

type UniversalVerifyRequest struct {
	Method string `json:"method"` // "2fa" or "passkey"
	Code   string `json:"code,omitempty"`
}

type VerificationStatusResponse struct {
	Verified  bool  `json:"verified"`
	ExpiresAt int64 `json:"expires_at,omitempty"`
}

// UniversalVerify is the generic verification endpoint
// Supports 2FA and Passkey verification; on success records a timestamp in the session
func UniversalVerify(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Not logged in",
		})
		return
	}

	var req UniversalVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, fmt.Errorf("Invalid parameter: %v", err))
		return
	}

	// Get user info
	user := &model.User{Id: userId}
	if err := user.FillUserById(); err != nil {
		common.ApiError(c, fmt.Errorf("Failed to get user info: %v", err))
		return
	}

	if user.Status != common.UserStatusEnabled {
		common.ApiError(c, fmt.Errorf("This user has been disabled"))
		return
	}

	// Check the user's verification method
	twoFA, _ := model.GetTwoFAByUserId(userId)
	has2FA := twoFA != nil && twoFA.IsEnabled

	passkey, passkeyErr := model.GetPasskeyByUserID(userId)
	hasPasskey := passkeyErr == nil && passkey != nil

	if !has2FA && !hasPasskey {
		common.ApiError(c, fmt.Errorf("The user has not enabled 2FA or Passkey"))
		return
	}

	// Verify according to the verification method
	var verified bool
	var verifyMethod string
	var err error

	switch req.Method {
	case "2fa":
		if !has2FA {
			common.ApiError(c, fmt.Errorf("2FA is not enabled for this user"))
			return
		}
		if req.Code == "" {
			common.ApiError(c, fmt.Errorf("Verification code cannot be empty"))
			return
		}
		verified = validateTwoFactorAuth(twoFA, req.Code)
		verifyMethod = "2FA"

	case "passkey":
		if !hasPasskey {
			common.ApiError(c, fmt.Errorf("Passkey is not enabled for this user"))
			return
		}
		// Passkey branch only trusts the short-lived marker written by PasskeyVerifyFinish.
		verified, err = consumePasskeyReady(c)
		if err != nil {
			common.ApiError(c, fmt.Errorf("Abnormal Passkey verification state: %v", err))
			return
		}
		if !verified {
			common.ApiError(c, fmt.Errorf("Please complete Passkey verification first"))
			return
		}
		verifyMethod = "Passkey"

	default:
		common.ApiError(c, fmt.Errorf("Unsupported verification method: %s", req.Method))
		return
	}

	if !verified {
		common.ApiError(c, fmt.Errorf("Verification failed, please check the code"))
		return
	}

	// Verification succeeded; record the timestamp in the session
	now, err := setSecureVerificationSession(c, req.Method)
	if err != nil {
		common.ApiError(c, fmt.Errorf("Failed to save verification state: %v", err))
		return
	}

	// Record the log
	model.RecordLog(userId, model.LogTypeSystem, fmt.Sprintf("Generic security verification succeeded (method: %s)", verifyMethod))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Verification successful",
		"data": gin.H{
			"verified":   true,
			"expires_at": now + SecureVerificationTimeout,
		},
	})
}

func setSecureVerificationSession(c *gin.Context, method string) (int64, error) {
	session := sessions.Default(c)
	session.Delete(PasskeyReadySessionKey)
	now := time.Now().Unix()
	session.Set(SecureVerificationSessionKey, now)
	session.Set(secureVerificationMethodSessionKey, method)
	if err := session.Save(); err != nil {
		return 0, err
	}
	return now, nil
}

func consumePasskeyReady(c *gin.Context) (bool, error) {
	session := sessions.Default(c)
	readyAtRaw := session.Get(PasskeyReadySessionKey)
	if readyAtRaw == nil {
		return false, nil
	}

	readyAt, ok := readyAtRaw.(int64)
	if !ok {
		session.Delete(PasskeyReadySessionKey)
		_ = session.Save()
		return false, fmt.Errorf("Invalid Passkey verification state")
	}
	session.Delete(PasskeyReadySessionKey)
	if err := session.Save(); err != nil {
		return false, err
	}
	// Expired ready markers cannot be reused.
	if time.Now().Unix()-readyAt >= PasskeyReadyTimeout {
		return false, nil
	}
	return true, nil
}
