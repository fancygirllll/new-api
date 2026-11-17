package common

import (
	"crypto/rand"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	// Backup code configuration
	BackupCodeLength = 8 // Backup code length
	BackupCodeCount  = 4 // Number of backup codes to generate

	// Limit configuration
	MaxFailAttempts = 5   // Maximum number of failed attempts
	LockoutDuration = 300 // Lockout duration (seconds)
)

// GenerateTOTPSecret generates a TOTP secret and configuration
func GenerateTOTPSecret(accountName string) (*otp.Key, error) {
	issuer := Get2FAIssuer()
	return totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
}

// ValidateTOTPCode validates a TOTP code
func ValidateTOTPCode(secret, code string) bool {
	// Clean up the code format
	cleanCode := strings.ReplaceAll(code, " ", "")
	if len(cleanCode) != 6 {
		return false
	}

	// Validate the code
	return totp.Validate(cleanCode, secret)
}

// GenerateBackupCodes generates backup recovery codes
func GenerateBackupCodes() ([]string, error) {
	codes := make([]string, BackupCodeCount)

	for i := 0; i < BackupCodeCount; i++ {
		code, err := generateRandomBackupCode()
		if err != nil {
			return nil, err
		}
		codes[i] = code
	}

	return codes, nil
}

// generateRandomBackupCode generates a single backup code
func generateRandomBackupCode() (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, BackupCodeLength)

	for i := range code {
		randomBytes := make([]byte, 1)
		_, err := rand.Read(randomBytes)
		if err != nil {
			return "", err
		}
		code[i] = charset[int(randomBytes[0])%len(charset)]
	}

	// Format as XXXX-XXXX
	return fmt.Sprintf("%s-%s", string(code[:4]), string(code[4:])), nil
}

// ValidateBackupCode validates the backup code format
func ValidateBackupCode(code string) bool {
	// Remove all separators and convert to upper case
	cleanCode := strings.ToUpper(strings.ReplaceAll(code, "-", ""))
	if len(cleanCode) != BackupCodeLength {
		return false
	}

	// Check whether the characters are valid
	for _, char := range cleanCode {
		if !((char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return false
		}
	}

	return true
}

// NormalizeBackupCode normalizes the backup code format
func NormalizeBackupCode(code string) string {
	cleanCode := strings.ToUpper(strings.ReplaceAll(code, "-", ""))
	if len(cleanCode) == BackupCodeLength {
		return fmt.Sprintf("%s-%s", cleanCode[:4], cleanCode[4:])
	}
	return code
}

// HashBackupCode hashes a backup code
func HashBackupCode(code string) (string, error) {
	normalizedCode := NormalizeBackupCode(code)
	return Password2Hash(normalizedCode)
}

// Get2FAIssuer returns the 2FA issuer name
func Get2FAIssuer() string {
	return SystemName
}

// getEnvOrDefault returns an environment variable or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// ValidateNumericCode validates the numeric code format
func ValidateNumericCode(code string) (string, error) {
	// Remove whitespace
	code = strings.ReplaceAll(code, " ", "")

	if len(code) != 6 {
		return "", fmt.Errorf("verification code must be 6 digits")
	}

	// Check whether it contains only digits
	if _, err := strconv.Atoi(code); err != nil {
		return "", fmt.Errorf("verification code may contain digits only")
	}

	return code, nil
}

// GenerateQRCodeData generates QR code data
func GenerateQRCodeData(secret, username string) string {
	issuer := Get2FAIssuer()
	accountName := fmt.Sprintf("%s (%s)", username, issuer)
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&digits=6&period=30",
		issuer, accountName, secret, issuer)
}
