package main

import (
	"log"

	"github.com/example/wg-panel/config"
	"github.com/example/wg-panel/database"
	"github.com/example/wg-panel/routes"
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

	r := routes.Setup(cfg)

	addr := ":" + cfg.ServerPort
	log.Printf("WireGuard panel listening on %s (swagger: http://localhost%s/swagger/index.html)", addr, addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
