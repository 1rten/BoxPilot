package middleware

import (
	"log"
	"net/http"

	"boxpilot/server/internal/api/dto"
	"boxpilot/server/internal/util/errorx"

	"github.com/gin-gonic/gin"
)

// Recover recovers from panics and returns 500.
func Recover() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %v", err)
				if appErr, ok := err.(*errorx.AppError); ok {
					c.JSON(appErr.HTTPStatus(), dto.ErrorEnvelope{
						Error: dto.ErrorObject{
							Code:    appErr.Code,
							Message: appErr.Message,
							Details: appErr.Details,
						},
					})
					return
				}
				c.JSON(http.StatusInternalServerError, dto.ErrorEnvelope{
					Error: dto.ErrorObject{Code: errorx.InternalError, Message: "internal error"},
				})
			}
		}()
		c.Next()
	}
}
