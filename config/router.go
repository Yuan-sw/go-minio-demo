package config

import (
	"minio-demo/service"

	"github.com/gin-gonic/gin"
)

func Routers() *gin.Engine {
	Routers := gin.Default()
	Routers.POST("/upload", service.Upload)
	Routers.GET("/getFile", service.GetFile)
	Routers.POST("/uploadPart", service.UploadPart)
	Routers.POST("/spiltBigFile", service.SpiltBigFile)
	return Routers
}
