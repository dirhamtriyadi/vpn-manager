package routes

import (
	"net/http"
	"strings"
	"time"

	"github.com/example/wg-panel/config"
	_ "github.com/example/wg-panel/docs"
	"github.com/example/wg-panel/handlers"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Setup builds the Gin engine with all routes and middleware.
func Setup(cfg *config.Config) *gin.Engine {
	r := gin.Default()

	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	if strings.TrimSpace(cfg.CORSAllowOrigins) == "*" {
		corsConfig.AllowAllOrigins = true
	} else {
		corsConfig.AllowOrigins = strings.Split(cfg.CORSAllowOrigins, ",")
	}
	r.Use(cors.New(corsConfig))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	ifaceHandler := handlers.NewInterfaceHandler(cfg)
	peerHandler := handlers.NewPeerHandler()

	api := r.Group("/api/v1")
	{
		ifaces := api.Group("/interfaces")
		{
			ifaces.GET("", ifaceHandler.List)
			ifaces.POST("", ifaceHandler.Create)
			ifaces.GET("/trash", ifaceHandler.Trash)
			ifaces.GET("/:id", ifaceHandler.Get)
			ifaces.PUT("/:id", ifaceHandler.Update)
			ifaces.DELETE("/:id", ifaceHandler.Delete)
			ifaces.POST("/:id/restore", ifaceHandler.Restore)
			ifaces.DELETE("/:id/purge", ifaceHandler.Purge)
			ifaces.POST("/:id/sync", ifaceHandler.Sync)
			ifaces.GET("/:id/status", ifaceHandler.Status)

			ifaces.GET("/:id/peers", peerHandler.List)
			ifaces.POST("/:id/peers", peerHandler.Create)
		}

		peers := api.Group("/peers")
		{
			peers.GET("/trash", peerHandler.Trash)
			peers.PUT("/:peerId", peerHandler.Update)
			peers.DELETE("/:peerId", peerHandler.Delete)
			peers.POST("/:peerId/restore", peerHandler.Restore)
			peers.DELETE("/:peerId/purge", peerHandler.Purge)
			peers.GET("/:peerId/config", peerHandler.Config)
			peers.GET("/:peerId/qrcode", peerHandler.QRCode)
		}
	}

	return r
}
