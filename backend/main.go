package main

import (
	"log"

	"github.com/example/vpn-manager/config"
	"github.com/example/vpn-manager/database"
	"github.com/example/vpn-manager/handlers"
	"github.com/example/vpn-manager/rbac"
	"github.com/example/vpn-manager/routes"
	"github.com/gin-gonic/gin"
)

// @title           VPN Manager API
// @version         1.0
// @description     Manage a WireGuard VPN concentrator (interfaces & peers) without touching the CLI.
// @description     Keys, client configs and QR codes are generated server-side; peers are pushed to the kernel via netlink.

// @contact.name   API Support
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1
// @schemes   http https

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer" followed by a space and the token from /auth/login.
func main() {
	cfg := config.Load()

	gin.SetMode(cfg.GinMode)

	database.Connect(cfg)

	// Seed the permission catalog + default roles and bootstrap the first
	// super-admin from AUTH_USERNAME/AUTH_PASSWORD when no users exist yet.
	if err := rbac.Seed(database.DB, cfg.AuthUsername, cfg.AuthPassword); err != nil {
		log.Fatalf("failed to seed RBAC: %v", err)
	}

	// Restore runtime state on startup off the serve path: bring enabled
	// interfaces back up and re-apply port-forward firewall rules (so they
	// survive a host reboot). Run it in a goroutine with a panic guard so a slow
	// or blocked netlink/iptables call can never delay — or prevent — the HTTP
	// server from listening. (A populated DB makes this do real kernel work at
	// boot; an empty one makes it a no-op, which is why an inline version only
	// ever hung after the volume had persisted state.)
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("bootstrap runtime panicked (ignored, API still serving): %v", rec)
			}
		}()
		handlers.BootstrapRuntime()
	}()

	r := routes.Setup(cfg)

	addr := ":" + cfg.ServerPort
	log.Printf("VPN Manager listening on %s (swagger: http://localhost%s/swagger/index.html)", addr, addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
