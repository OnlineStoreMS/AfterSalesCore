package admin

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	g *gin.RouterGroup,
	unboxingH *UnboxingHandler,
	edgeRecordH *EdgeRecordHandler,
	edgeDeviceH *EdgeDeviceHandler,
	shopH *ShopHandler,
	notifyH *NotificationHandler,
) {
	// legacy browser unboxing (kept for backward compatibility)
	g.GET("/unboxing-records", unboxingH.List)
	g.POST("/unboxing-records", unboxingH.Create)
	g.GET("/unboxing-records/:id", unboxingH.Get)
	g.POST("/unboxing-records/:id/video", unboxingH.UploadVideo)
	g.POST("/unboxing-records/:id/photos", unboxingH.UploadPhoto)
	g.POST("/unboxing-records/:id/complete", unboxingH.Complete)
	g.GET("/unboxing-records/:id/video/download", unboxingH.VideoDownload)

	// unified edge records (BoxEdge sync + cloud browser)
	g.GET("/edge-records/stats", edgeRecordH.Stats)
	g.GET("/edge-records", edgeRecordH.List)
	g.POST("/edge-records", edgeRecordH.Create)
	g.GET("/edge-records/:id", edgeRecordH.Get)
	g.POST("/edge-records/:id/video", edgeRecordH.UploadVideo)
	g.POST("/edge-records/:id/photos", edgeRecordH.UploadPhoto)
	g.POST("/edge-records/:id/complete", edgeRecordH.Complete)
	g.DELETE("/edge-records/:id", edgeRecordH.Delete)
	g.POST("/edge-records/batch-delete", edgeRecordH.BatchDelete)
	g.GET("/edge-records/:id/video/download", edgeRecordH.VideoDownload)

	// edge device registry
	g.GET("/edge-devices", edgeDeviceH.List)
	g.POST("/edge-devices", edgeDeviceH.Create)
	g.POST("/edge-devices/sync", edgeDeviceH.Sync)
	g.PUT("/edge-devices/:id", edgeDeviceH.Update)
	g.DELETE("/edge-devices/:id", edgeDeviceH.Delete)
	g.POST("/edge-devices/:id/probe", edgeDeviceH.Probe)

	g.GET("/shops", shopH.List)
	g.POST("/shops", shopH.Create)
	g.GET("/shops/:id", shopH.Get)
	g.PUT("/shops/:id", shopH.Update)
	g.DELETE("/shops/:id", shopH.Delete)
	g.POST("/shops/:id/reset-bind", shopH.ResetBind)
	g.POST("/shops/:id/request-sync", shopH.RequestSync)
	g.GET("/shops/:id/workbench", shopH.Workbench)
	g.GET("/shops/:id/tickets", shopH.Tickets)

	g.GET("/notifications", notifyH.Get)
	g.PUT("/notifications", notifyH.Save)
	g.POST("/notifications/test", notifyH.Test)
	g.POST("/notifications/test-barcode", notifyH.TestBarcode)
	g.POST("/notifications/run", notifyH.Run)
	g.POST("/notifications/reset-state", notifyH.ResetState)
}
