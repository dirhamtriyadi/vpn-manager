package routes

import (
	"strings"
	"time"

	"github.com/example/wg-panel/auth"
	"github.com/example/wg-panel/config"
	"github.com/example/wg-panel/dto"
	_ "github.com/example/wg-panel/docs"
	"github.com/example/wg-panel/handlers"
	"github.com/example/wg-panel/middleware"
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
		dto.OK(c, "health check passed", gin.H{"status": "ok"})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	authSvc := auth.NewService(
		cfg.AuthUsername,
		cfg.AuthPassword,
		cfg.AuthTokenSecret,
		time.Duration(cfg.AuthTokenTTLHours)*time.Hour,
	)
	authHandler := handlers.NewAuthHandler(authSvc)
	ifaceHandler := handlers.NewInterfaceHandler(cfg)
	peerHandler := handlers.NewPeerHandler()
	vpnHandler := handlers.NewVPNHandler()
	openVPNHandler := handlers.NewOpenVPNHandler()

	api := r.Group("/api/v1")
	{
		// Public: obtain a token with the env-configured credentials.
		api.POST("/auth/login", authHandler.Login)

		// Everything below requires a valid bearer token.
		protected := api.Group("")
		protected.Use(middleware.Auth(authSvc))
		protected.GET("/auth/me", authHandler.Me)

		ifaces := protected.Group("/interfaces")
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

		peers := protected.Group("/peers")
		{
			peers.GET("/trash", peerHandler.Trash)
			peers.PUT("/:peerId", peerHandler.Update)
			peers.DELETE("/:peerId", peerHandler.Delete)
			peers.POST("/:peerId/restore", peerHandler.Restore)
			peers.DELETE("/:peerId/purge", peerHandler.Purge)
			peers.GET("/:peerId/config", peerHandler.Config)
			peers.GET("/:peerId/qrcode", peerHandler.QRCode)
		}

		vpn := protected.Group("/vpn")
		{
			vpn.GET("/protocols", vpnHandler.Protocols)
			vpn.GET("/roadmaps/:protocol", vpnHandler.ProtocolRoadmap)
			vpn.GET("/service-plans/:protocol", vpnHandler.ProtocolServicePlan)
			vpn.GET("/instances", vpnHandler.Instances)
			vpn.GET("/instances/:id", vpnHandler.Instance)
			vpn.GET("/instances/:id/users", vpnHandler.Users)
			vpn.GET("/instances/:id/status", vpnHandler.Status)

			openvpn := vpn.Group("/openvpn")
			{
				openvpn.GET("/roadmap", openVPNHandler.Roadmap)
				openvpn.GET("/instances", openVPNHandler.ListInstances)
				openvpn.POST("/instances", openVPNHandler.CreateInstanceDraft)
				openvpn.GET("/instances/:id/runtime-manifest", openVPNHandler.GetRuntimeManifest)
				openvpn.POST("/instances/:id/runtime-manifest", openVPNHandler.GenerateRuntimeManifest)
				openvpn.GET("/instances/:id/users", openVPNHandler.ListUsers)
				openvpn.POST("/instances/:id/users", openVPNHandler.CreateUserDraft)
				openvpn.POST("/instances/:id/lifecycle/:action", openVPNHandler.LifecyclePlan)
				openvpn.POST("/instances/:id/firewall-plan", openVPNHandler.FirewallPlan)
				openvpn.POST("/status/parse", openVPNHandler.ParseStatus)
				openvpn.POST("/runtime/preview", openVPNHandler.PreviewRuntimeManifest)
				openvpn.POST("/client-profile/preview", openVPNHandler.PreviewClientProfile)
			}
		}
	}

	return r
}
