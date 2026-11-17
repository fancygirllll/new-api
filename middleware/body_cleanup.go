package middleware

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// BodyStorageCleanup is the middleware that cleans up request body storage.
// It automatically cleans up disk/memory caches after the request has been handled.
func BodyStorageCleanup() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Handle the request.
		c.Next()

		// Clean up storage after the request finishes.
		common.CleanupBodyStorage(c)

		// Clean up file caches (such as files downloaded from URLs).
		service.CleanupFileSources(c)
	}
}
