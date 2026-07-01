package admin

import "github.com/gin-gonic/gin"

func RegisterRoutes(g *gin.RouterGroup, unboxingH *UnboxingHandler) {
	g.GET("/unboxing-records", unboxingH.List)
	g.POST("/unboxing-records", unboxingH.Create)
	g.GET("/unboxing-records/:id", unboxingH.Get)
	g.POST("/unboxing-records/:id/video", unboxingH.UploadVideo)
	g.POST("/unboxing-records/:id/photos", unboxingH.UploadPhoto)
	g.POST("/unboxing-records/:id/complete", unboxingH.Complete)
	g.GET("/unboxing-records/:id/video/download", unboxingH.VideoDownload)
}
