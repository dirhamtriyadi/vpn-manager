package routes

import (
	"strings"
	"time"

	"github.com/example/vpn-manager/auth"
	"github.com/example/vpn-manager/config"
	_ "github.com/example/vpn-manager/docs"
	"github.com/example/vpn-manager/dto"
	"github.com/example/vpn-manager/handlers"
	"github.com/example/vpn-manager/middleware"
	"github.com/example/vpn-manager/rbac"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// rp is a short alias for the permission-guard middleware.
var rp = middleware.RequirePermission

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
		cfg.AuthTokenSecret,
		time.Duration(cfg.AuthTokenTTLHours)*time.Hour,
	)
	authHandler := handlers.NewAuthHandler(authSvc)
	ifaceHandler := handlers.NewInterfaceHandler(cfg)
	peerHandler := handlers.NewPeerHandler()
	vpnHandler := handlers.NewVPNHandler()
	openVPNHandler := handlers.NewOpenVPNHandler()
	userHandler := handlers.NewUserHandler()
	roleHandler := handlers.NewRoleHandler()
	permissionHandler := handlers.NewPermissionHandler()
	portForwardHandler := handlers.NewPortForwardHandler(cfg)

	api := r.Group("/api/v1")
	{
		// Public: exchange credentials for a token.
		api.POST("/auth/login", authHandler.Login)

		// Everything below requires a valid bearer token. Per-route guards add
		// the required permission on top.
		protected := api.Group("")
		protected.Use(middleware.Auth(authSvc))
		protected.GET("/auth/me", authHandler.Me)

		ifaces := protected.Group("/interfaces")
		{
			ifaces.GET("", rp(rbac.PermInterfacesView), ifaceHandler.List)
			ifaces.POST("", rp(rbac.PermInterfacesCreate), ifaceHandler.Create)
			ifaces.GET("/trash", rp(rbac.PermInterfacesView), ifaceHandler.Trash)
			ifaces.GET("/:id", rp(rbac.PermInterfacesView), ifaceHandler.Get)
			ifaces.PUT("/:id", rp(rbac.PermInterfacesUpdate), ifaceHandler.Update)
			ifaces.DELETE("/:id", rp(rbac.PermInterfacesDelete), ifaceHandler.Delete)
			ifaces.POST("/:id/restore", rp(rbac.PermInterfacesDelete), ifaceHandler.Restore)
			ifaces.DELETE("/:id/purge", rp(rbac.PermInterfacesDelete), ifaceHandler.Purge)
			ifaces.POST("/:id/sync", rp(rbac.PermInterfacesSync), ifaceHandler.Sync)
			ifaces.GET("/:id/status", rp(rbac.PermInterfacesView), ifaceHandler.Status)

			ifaces.GET("/:id/peers", rp(rbac.PermPeersView), peerHandler.List)
			ifaces.POST("/:id/peers", rp(rbac.PermPeersCreate), peerHandler.Create)
		}

		// Public-IP port forwarding: expose a server WAN port into a peer's tunnel.
		portForwards := protected.Group("/port-forwards")
		{
			portForwards.GET("", rp(rbac.PermPortForwardsView), portForwardHandler.List)
			portForwards.POST("", rp(rbac.PermPortForwardsManage), portForwardHandler.Create)
			portForwards.GET("/:id", rp(rbac.PermPortForwardsView), portForwardHandler.Get)
			portForwards.PUT("/:id", rp(rbac.PermPortForwardsManage), portForwardHandler.Update)
			portForwards.DELETE("/:id", rp(rbac.PermPortForwardsManage), portForwardHandler.Delete)
		}

		peers := protected.Group("/peers")
		{
			peers.GET("/trash", rp(rbac.PermPeersView), peerHandler.Trash)
			peers.PUT("/:peerId", rp(rbac.PermPeersUpdate), peerHandler.Update)
			peers.DELETE("/:peerId", rp(rbac.PermPeersDelete), peerHandler.Delete)
			peers.POST("/:peerId/restore", rp(rbac.PermPeersDelete), peerHandler.Restore)
			peers.DELETE("/:peerId/purge", rp(rbac.PermPeersDelete), peerHandler.Purge)
			peers.GET("/:peerId/config", rp(rbac.PermPeersView), peerHandler.Config)
			peers.GET("/:peerId/qrcode", rp(rbac.PermPeersView), peerHandler.QRCode)
		}

		vpn := protected.Group("/vpn")
		{
			vpn.GET("/protocols", rp(rbac.PermVPNView), vpnHandler.Protocols)
			vpn.GET("/roadmaps/:protocol", rp(rbac.PermVPNView), vpnHandler.ProtocolRoadmap)
			vpn.GET("/service-plans/:protocol", rp(rbac.PermVPNView), vpnHandler.ProtocolServicePlan)
			vpn.GET("/production-plans/:protocol", rp(rbac.PermVPNView), vpnHandler.ProtocolProductionPlan)
			vpn.POST("/config-preview", rp(rbac.PermVPNView), vpnHandler.PreviewProtocolConfig)
			vpn.GET("/:protocol/instances", rp(rbac.PermVPNView), vpnHandler.ListLegacyInstances)
			vpn.POST("/:protocol/instances", rp(rbac.PermVPNCreate), vpnHandler.CreateLegacyInstanceDraft)
			vpn.POST("/:protocol/instances/:id/apply", rp(rbac.PermVPNApply), vpnHandler.ApplyLegacyInstance)
			vpn.GET("/instances", rp(rbac.PermVPNView), vpnHandler.Instances)
			vpn.GET("/instances/:id", rp(rbac.PermVPNView), vpnHandler.Instance)
			vpn.GET("/instances/:id/users", rp(rbac.PermVPNView), vpnHandler.Users)
			vpn.GET("/instances/:id/status", rp(rbac.PermVPNView), vpnHandler.Status)

			openvpn := vpn.Group("/openvpn")
			{
				openvpn.GET("/roadmap", rp(rbac.PermOpenVPNView), openVPNHandler.Roadmap)
				openvpn.GET("/instances", rp(rbac.PermOpenVPNView), openVPNHandler.ListInstances)
				openvpn.POST("/instances", rp(rbac.PermOpenVPNCreate), openVPNHandler.CreateInstanceDraft)
				openvpn.GET("/instances/:id/runtime-manifest", rp(rbac.PermOpenVPNView), openVPNHandler.GetRuntimeManifest)
				openvpn.POST("/instances/:id/runtime-manifest", rp(rbac.PermOpenVPNCreate), openVPNHandler.GenerateRuntimeManifest)
				openvpn.GET("/instances/:id/users", rp(rbac.PermOpenVPNView), openVPNHandler.ListUsers)
				openvpn.POST("/instances/:id/users", rp(rbac.PermOpenVPNCreate), openVPNHandler.CreateUserDraft)
				openvpn.POST("/instances/:id/lifecycle/:action", rp(rbac.PermOpenVPNView), openVPNHandler.LifecyclePlan)
				openvpn.POST("/instances/:id/firewall-plan", rp(rbac.PermOpenVPNView), openVPNHandler.FirewallPlan)
				openvpn.POST("/instances/:id/apply", rp(rbac.PermOpenVPNApply), openVPNHandler.ApplyRuntime)
				openvpn.POST("/status/parse", rp(rbac.PermOpenVPNView), openVPNHandler.ParseStatus)
				openvpn.POST("/runtime/preview", rp(rbac.PermOpenVPNView), openVPNHandler.PreviewRuntimeManifest)
				openvpn.POST("/client-profile/preview", rp(rbac.PermOpenVPNView), openVPNHandler.PreviewClientProfile)
			}
		}

		// RBAC administration.
		users := protected.Group("/users")
		{
			users.GET("", rp(rbac.PermUsersView), userHandler.List)
			users.POST("", rp(rbac.PermUsersCreate), userHandler.Create)
			users.GET("/:id", rp(rbac.PermUsersView), userHandler.Get)
			users.PUT("/:id", rp(rbac.PermUsersUpdate), userHandler.Update)
			users.DELETE("/:id", rp(rbac.PermUsersDelete), userHandler.Delete)
			users.PUT("/:id/roles", rp(rbac.PermUsersUpdate), userHandler.SetRoles)
			users.PUT("/:id/permissions", rp(rbac.PermUsersUpdate), userHandler.SetPermissions)
		}

		roles := protected.Group("/roles")
		{
			roles.GET("", rp(rbac.PermRolesView), roleHandler.List)
			roles.POST("", rp(rbac.PermRolesManage), roleHandler.Create)
			roles.GET("/:id", rp(rbac.PermRolesView), roleHandler.Get)
			roles.PUT("/:id", rp(rbac.PermRolesManage), roleHandler.Update)
			roles.DELETE("/:id", rp(rbac.PermRolesManage), roleHandler.Delete)
			roles.PUT("/:id/permissions", rp(rbac.PermRolesManage), roleHandler.SetPermissions)
		}

		protected.GET("/permissions", rp(rbac.PermPermissionsView), permissionHandler.List)
	}

	return r
}
