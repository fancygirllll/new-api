package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// Setup2FARequest is the 2FA setup request structure
type Setup2FARequest struct {
	Code string `json:"code" binding:"required"`
}

// Verify2FARequest is the 2FA verification request structure
type Verify2FARequest struct {
	Code string `json:"code" binding:"required"`
}

// Setup2FAResponse is the 2FA setup response structure
type Setup2FAResponse struct {
	Secret      string   `json:"secret"`
	QRCodeData  string   `json:"qr_code_data"`
	BackupCodes []string `json:"backup_codes"`
}

// Setup2FA initializes 2FA setup
func Setup2FA(c *gin.Context) {
	userId := c.GetInt("id")

	// Check whether the user has already enabled 2FA
	existing, err := model.GetTwoFAByUserId(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if existing != nil && existing.IsEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "2FA is already enabled for this user. Disable it before setting up again",
		})
		return
	}

	// If a disabled 2FA record exists, delete it first
	if existing != nil && !existing.IsEnabled {
		if err := existing.Delete(); err != nil {
			common.ApiError(c, err)
			return
		}
		existing = nil // Reset to nil; a new record will be created later
	}

	// Get user info
	user, err := model.GetUserById(userId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// Generate the TOTP secret
	key, err := common.GenerateTOTPSecret(user.Username)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Failed to generate the 2FA secret",
		})
		common.SysLog("Failed to generate the TOTP secret: " + err.Error())
		return
	}

	// Generate backup codes
	backupCodes, err := common.GenerateBackupCodes()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Failed to generate backup codes",
		})
		common.SysLog("Failed to generate backup codes: " + err.Error())
		return
	}

	// Generate QR code data
	qrCodeData := common.GenerateQRCodeData(key.Secret(), user.Username)

	// Create or update the 2FA record (not enabled yet)
	twoFA := &model.TwoFA{
		UserId:    userId,
		Secret:    key.Secret(),
		IsEnabled: false,
	}

	if existing != nil {
		// Update the existing record
		twoFA.Id = existing.Id
		err = twoFA.Update()
	} else {
		// Create a new record
		err = twoFA.Create()
	}

	if err != nil {
		common.ApiError(c, err)
		return
	}

	// Create backup code records
	if err := model.CreateBackupCodes(userId, backupCodes); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Failed to save backup codes",
		})
		common.SysLog("Failed to save backup codes: " + err.Error())
		return
	}

	// Record the operation log
	model.RecordLog(userId, model.LogTypeSystem, "Started setting up two-factor authentication")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "2FA setup initialized. Scan the QR code with your authenticator app and enter the verification code to complete setup",
		"data": Setup2FAResponse{
			Secret:      key.Secret(),
			QRCodeData:  qrCodeData,
			BackupCodes: backupCodes,
		},
	})
}

// Enable2FA enables 2FA
func Enable2FA(c *gin.Context) {
	var req Setup2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Invalid parameter",
		})
		return
	}

	userId := c.GetInt("id")

	// Get the 2FA record
	twoFA, err := model.GetTwoFAByUserId(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if twoFA == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Please complete the 2FA initial setup first",
		})
		return
	}
	if twoFA.IsEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "2FA is already enabled",
		})
		return
	}

	// Verify the TOTP code
	cleanCode, err := common.ValidateNumericCode(req.Code)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	if !common.ValidateTOTPCode(twoFA.Secret, cleanCode) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Invalid verification code or backup code, please try again",
		})
		return
	}

	// Enable 2FA
	if err := twoFA.Enable(); err != nil {
		common.ApiError(c, err)
		return
	}

	// Record the operation log
	model.RecordLog(userId, model.LogTypeSystem, "Two-factor authentication enabled")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Two-factor authentication enabled successfully",
	})
}

// Disable2FA disables 2FA
func Disable2FA(c *gin.Context) {
	var req Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Invalid parameter",
		})
		return
	}

	userId := c.GetInt("id")

	// Get the 2FA record
	twoFA, err := model.GetTwoFAByUserId(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if twoFA == nil || !twoFA.IsEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "2FA is not enabled for this user",
		})
		return
	}

	// Verify the TOTP code or a backup code
	cleanCode, err := common.ValidateNumericCode(req.Code)
	isValidTOTP := false
	isValidBackup := false

	if err == nil {
		// Try TOTP verification
		isValidTOTP, _ = twoFA.ValidateTOTPAndUpdateUsage(cleanCode)
	}

	if !isValidTOTP {
		// Try backup code verification
		isValidBackup, err = twoFA.ValidateBackupCodeAndUpdateUsage(req.Code)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	}

	if !isValidTOTP && !isValidBackup {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Invalid verification code or backup code, please try again",
		})
		return
	}

	// Disable 2FA
	if err := model.DisableTwoFA(userId); err != nil {
		common.ApiError(c, err)
		return
	}

	// Record the operation log
	model.RecordLog(userId, model.LogTypeSystem, "Disabled two-factor authentication")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Two-factor authentication has been disabled",
	})
}

// Get2FAStatus returns the user's 2FA status
func Get2FAStatus(c *gin.Context) {
	userId := c.GetInt("id")

	twoFA, err := model.GetTwoFAByUserId(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	status := map[string]interface{}{
		"enabled": false,
		"locked":  false,
	}

	if twoFA != nil {
		status["enabled"] = twoFA.IsEnabled
		status["locked"] = twoFA.IsLocked()
		if twoFA.IsEnabled {
			// Get the number of remaining backup codes
			backupCount, err := model.GetUnusedBackupCodeCount(userId)
			if err != nil {
				common.SysLog("Failed to get the backup code count: " + err.Error())
			} else {
				status["backup_codes_remaining"] = backupCount
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    status,
	})
}

// RegenerateBackupCodes regenerates backup codes
func RegenerateBackupCodes(c *gin.Context) {
	var req Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Invalid parameter",
		})
		return
	}

	userId := c.GetInt("id")

	// Get the 2FA record
	twoFA, err := model.GetTwoFAByUserId(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if twoFA == nil || !twoFA.IsEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "2FA is not enabled for this user",
		})
		return
	}

	// Verify the TOTP code
	cleanCode, err := common.ValidateNumericCode(req.Code)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	valid, err := twoFA.ValidateTOTPAndUpdateUsage(cleanCode)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if !valid {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Invalid verification code or backup code, please try again",
		})
		return
	}

	// Generate new backup codes
	backupCodes, err := common.GenerateBackupCodes()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Failed to generate backup codes",
		})
		common.SysLog("Failed to generate backup codes: " + err.Error())
		return
	}

	// Save the new backup codes
	if err := model.CreateBackupCodes(userId, backupCodes); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Failed to save backup codes",
		})
		common.SysLog("Failed to save backup codes: " + err.Error())
		return
	}

	// Record the operation log
	model.RecordLog(userId, model.LogTypeSystem, "Regenerated two-factor authentication backup codes")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Backup codes regenerated successfully",
		"data": map[string]interface{}{
			"backup_codes": backupCodes,
		},
	})
}

// Verify2FALogin verifies 2FA during login
func Verify2FALogin(c *gin.Context) {
	var req Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Invalid parameter",
		})
		return
	}

	// Get pending user info from the session
	session := sessions.Default(c)
	pendingUserId := session.Get("pending_user_id")
	if pendingUserId == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Session expired, please log in again",
		})
		return
	}
	userId, ok := pendingUserId.(int)
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Invalid session data, please log in again",
		})
		return
	}
	// Get user info
	user, err := model.GetUserById(userId, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "User not found",
		})
		return
	}

	// Get the 2FA record
	twoFA, err := model.GetTwoFAByUserId(user.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if twoFA == nil || !twoFA.IsEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "2FA is not enabled for this user",
		})
		return
	}

	// Verify the TOTP code or a backup code
	cleanCode, err := common.ValidateNumericCode(req.Code)
	isValidTOTP := false
	isValidBackup := false

	if err == nil {
		// Try TOTP verification
		isValidTOTP, _ = twoFA.ValidateTOTPAndUpdateUsage(cleanCode)
	}

	if !isValidTOTP {
		// Try backup code verification
		isValidBackup, err = twoFA.ValidateBackupCodeAndUpdateUsage(req.Code)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	}

	if !isValidTOTP && !isValidBackup {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Invalid verification code or backup code, please try again",
		})
		return
	}

	// 2FA verified; clear the pending session info and complete login
	session.Delete("pending_username")
	session.Delete("pending_user_id")
	session.Save()

	setupLogin(user, c)
}

// Admin2FAStats returns 2FA statistics for administrators
func Admin2FAStats(c *gin.Context) {
	stats, err := model.GetTwoFAStats()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    stats,
	})
}

// AdminDisable2FA force-disables a user's 2FA as an administrator
func AdminDisable2FA(c *gin.Context) {
	userIdStr := c.Param("id")
	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Invalid user ID format",
		})
		return
	}

	// Check the target user's permissions
	targetUser, err := model.GetUserById(userId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	myRole := c.GetInt("role")
	if !canManageTargetRole(myRole, targetUser.Role) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "You cannot manage 2FA settings for users with equal or higher privileges",
		})
		return
	}

	// Disable 2FA
	if err := model.DisableTwoFA(userId); err != nil {
		if errors.Is(err, model.ErrTwoFANotEnabled) {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "2FA is not enabled for this user",
			})
			return
		}
		common.ApiError(c, err)
		return
	}

	recordManageAuditFor(c, userId, "user.2fa_disable", nil)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "2FA has been force-disabled for this user",
	})
}
