package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"go.opentelemetry.io/otel"
	_ "modernc.org/sqlite"

	useradapter "restful-boilerplate/adapter/user"
	"restful-boilerplate/domain/user"
	"restful-boilerplate/infra/config"
	router "restful-boilerplate/infra/http"
	"restful-boilerplate/infra/metrics"
	requestlogger "restful-boilerplate/infra/middleware"
	"restful-boilerplate/infra/otelhttp"
	infradb "restful-boilerplate/infra/sqlite"
	"restful-boilerplate/infra/telemetry"
	cv "restful-boilerplate/infra/validator"
)

// @title          Restful Boilerplate API
// @version        1.0
// @description    Go RESTful API boilerplate built on net/http + SQLite.
// @host           localhost:8080
// @BasePath       /api
// @schemes        http
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	cfg := config.Load(os.Getenv)
	stopTracing, err := telemetry.SetupAll(ctx, "./logs/app.log", cfg.LogFormat)

	if err != nil {
		slog.Error("failed to setup tracing", "error", err)
		stop()
		os.Exit(1)
	}

	db, err := infradb.OpenDB(ctx, "./data.db")
	if err != nil {
		slog.Error("failed to open database", "error", err)
		stopTracing()
		stop()
		os.Exit(1)
	}

	// All early-exit paths done; defers are safe from here.
	defer stop()
	defer stopTracing()
	defer db.Close() //nolint:errcheck // best-effort close on exit
	v := cv.New()

	r := router.NewRouter()
	r.Prefix("/api")
	r.Use(otelhttp.Middleware("restful-boilerplate"))
	r.Use(requestlogger.RequestLog)
	r.Use(requestlogger.Recovery)
	r.Route("GET /metrics", metrics.Handler())

	// User domain register
	userRepo := useradapter.NewSQLite(db)
	userSvc := user.NewService(userRepo, otel.Tracer("user"))
	r.Group("/users", useradapter.NewHandler(userSvc, v).RegisterRoutes)

	addr := net.JoinHostPort(cfg.Server.Host, cfg.Server.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      r.Handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	router.GracefulServe(ctx, httpServer, 10*time.Second)
}
