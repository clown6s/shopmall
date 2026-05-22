package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"shopmall/cmd/server"
	"shopmall/common/middleware"
	"shopmall/internal/config"
	"shopmall/pkg"
	"shopmall/routes"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// 1. Load config
	cfgPath := flag.String("config", "config.yaml", "config file path")
	flag.Parse()

	cfg := config.Load(*cfgPath)
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config validation failed: %v", err)
	}

	// 2. Init logger
	pkg.InitLogger(cfg.App.Mode)
	defer pkg.Sync()

	// 3. Set Gin mode
	switch cfg.App.Mode {
	case "prod":
		gin.SetMode(gin.ReleaseMode)
	case "dev":
		gin.SetMode(gin.DebugMode)
	default:
		gin.SetMode(gin.TestMode)
	}

	// 4. Wire init
	app, err := server.InitializeApp(cfg)
	if err != nil {
		log.Fatalf("init app failed: %v", err)
	}
	defer func() {
		if err := app.DB.Close(); err != nil {
			pkg.Logger.Error("close db failed", zap.Error(err))
		}
	}()
	if app.Redis != nil {
		defer app.Redis.Close()
	}

	// 5. Create Gin engine
	r := gin.New()

	// 6. Global middleware (outer -> inner)
	r.Use(
		middleware.TraceIDMiddleware(),
		middleware.SecurityHeadersMiddleware(),
		middleware.CORSMiddleware(middleware.NewCORSOptions(&cfg.CORS)),
		middleware.RecoveryMiddleware(),
	)

	// 7. Register routes
	hc := &routes.HealthChecker{DB: app.DB, Redis: app.Redis}
	routes.RegisterRoutes(r, hc)

	// 8. Start HTTP server
	addr := cfg.App.Port
	if len(addr) > 0 && addr[0] != ':' {
		addr = ":" + addr
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  90 * time.Second,
	}

	go func() {
		pkg.Logger.Info("server starting", zap.String("addr", "http://localhost"+addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			pkg.Logger.Error("server error", zap.Error(err))
			log.Fatalf("server error: %v", err)
		}
	}()

	// 9. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	pkg.Logger.Info("shutting down server...", zap.String("signal", sig.String()))

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(timeoutCtx); err != nil {
		pkg.Logger.Error("server shutdown forced", zap.Error(err))
	}

	pkg.Logger.Info("server stopped")
}
