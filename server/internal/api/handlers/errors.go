package handlers

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"boxpilot/server/internal/api/dto"
	"boxpilot/server/internal/util/errorx"
)

func writeError(c *gin.Context, err *errorx.AppError) {
	c.JSON(err.HTTPStatus(), dto.ErrorEnvelope{
		Error: dto.ErrorObject{
			Code:    err.Code,
			Message: err.Message,
			Details: err.Details,
		},
	})
}
