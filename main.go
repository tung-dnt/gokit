package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.opentelemetry.io/otel"

	_ "restful-boilerplate/docs"
	"restful-boilerplate/internal/app"
	"restful-boilerplate/internal/user"
	"restful-boilerplate/pkg/config"
	router "restful-boilerplate/pkg/http"
	"restful-boilerplate/pkg/logger"
	"restful-boilerplate/pkg/metrics"
	"restful-boilerplate/pkg/otelhttp"
	"restful-boilerplate/pkg/postgres"
	pgdb "restful-boilerplate/pkg/postgres/db"
	"restful-boilerplate/pkg/recovery"
	"restful-boilerplate/pkg/telemetry"
	cv "restful-boilerplate/pkg/validator"
)

// @title          Restful Boilerplate API
// @version        1.0
// @description    Go RESTful API boilerplate built on net/http + PostgreSQL.
// @host           localhost:4040
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

	pool, err := postgres.OpenDB(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		stopTracing()
		stop()
		os.Exit(1)
	}

	if err = postgres.Migrate(ctx, pool); err != nil {
		slog.Error("failed to migrate database", "error", err)
		stopTracing()
		pool.Close()
		stop()
		os.Exit(1)
	}

	// All early-exit paths done; defers are safe from here.
	defer stop()
	defer stopTracing()
	defer pool.Close()

	v := cv.New()
	metric := metrics.New()

	a := &app.App{
		Queries:   pgdb.New(pool),
		Validator: v,
		Tracer:    otel.GetTracerProvider(),
	}

	r := router.NewRouter()
	r.Use(metric.Middleware)
	r.Use(otelhttp.Middleware("restful-boilerplate"))
	r.Use(logger.Middleware)
	r.Use(recovery.Middleware)
	r.GET("/metrics", metric.Handler().ServeHTTP)

	r.Group("/v1", func(g *router.Group) {
		g.Prefix("/api")
		g.ANY("/swagger/", httpSwagger.WrapHandler)
		// User domain register
		g.Group("/users", user.NewModule(a).RegisterRoutes)
	})

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
