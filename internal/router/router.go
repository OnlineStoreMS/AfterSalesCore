package router

import (
	"context"
	"path/filepath"

	"aftersalescore/admin"
	adminmw "aftersalescore/admin/middleware"
	"aftersalescore/internal/config"
	jwtmgr "aftersalescore/internal/pkg/jwt"
	"aftersalescore/internal/repo"
	"aftersalescore/internal/scheduler"
	"aftersalescore/internal/service"
	"aftersalescore/internal/storage"
	"aftersalescore/plugin"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), corsMiddleware(cfg))

	if cfg.Storage.Driver == "local" || cfg.Storage.Driver == "" {
		uploadDir := filepath.Join(cfg.Storage.LocalPath, cfg.Storage.Prefix)
		r.Static("/uploads", uploadDir)
	}

	store, err := storage.New(&cfg.Storage)
	if err != nil {
		panic(err)
	}
	media := storage.NewEdgeMediaResolver(cfg, store)
	edgeObjects, err := storage.NewEdgeObjectStore(cfg, store)
	if err != nil {
		panic(err)
	}

	repos := repo.New(db)
	unboxingSvc := service.NewUnboxingService(repos, store)
	edgeRecordSvc := service.NewEdgeRecordService(repos, store, media, edgeObjects, cfg)
	edgeDeviceSvc := service.NewEdgeDeviceService(repos, cfg)
	_ = edgeDeviceSvc.EnsureDefaults()
	_ = edgeDeviceSvc.SyncFromRecords()

	shopSvc := service.NewShopService(repos)
	notifySvc := service.NewNotificationService(repos)
	unboxingH := admin.NewUnboxingHandler(unboxingSvc)
	edgeRecordH := admin.NewEdgeRecordHandler(edgeRecordSvc)
	edgeDeviceH := admin.NewEdgeDeviceHandler(edgeDeviceSvc)
	shopH := admin.NewShopHandler(shopSvc)
	notifyH := admin.NewNotificationHandler(notifySvc)
	pluginH := plugin.NewHandler(shopSvc, notifySvc)

	go edgeDeviceSvc.StartHealthPoller(context.Background())
	scheduler.NewNotificationScheduler(notifySvc).Start()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "aftersalescore"})
	})

	v1 := r.Group("/api/v1")
	adminGroup := v1.Group("/admin")
	jwtMgr := jwtmgr.NewManager(cfg.Auth.JWTSecret)
	adminGroup.Use(adminmw.AdminAuth(&cfg.Auth, jwtMgr))
	admin.RegisterRoutes(adminGroup, unboxingH, edgeRecordH, edgeDeviceH, shopH, notifyH)

	pluginGroup := v1.Group("/plugin")
	pluginGroup.POST("/bind", pluginH.Bind)
	authed := pluginGroup.Group("")
	authed.Use(pluginH.AuthRequired())
	authed.POST("/heartbeat", pluginH.Heartbeat)
	authed.POST("/sync", pluginH.Sync)

	return r
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	origins := cfg.CORS.AllowOrigins
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := origin == ""
		for _, o := range origins {
			if o == origin || o == "*" {
				allowed = true
				break
			}
		}
		if allowed && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Plugin-Key,X-Plugin-Secret")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
