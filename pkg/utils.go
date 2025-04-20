package internals

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const ContextLoggerKey = "contextLogger"

func GetLogger(c *gin.Context) *zap.Logger {
	if logger, exists := c.Get(ContextLoggerKey); exists {
		if zapLogger, ok := logger.(*zap.Logger); ok {
			return zapLogger
		}
	}
	return zap.NewNop() // fallback to no-op logger
}
