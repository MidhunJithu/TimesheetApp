package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func AbortWithError(ctx *gin.Context, err error) {
	logrus.Errorf("error happened : %s", err)
	ctx.JSON(http.StatusInternalServerError, gin.H{
		"data": "not ok",
	})
}
