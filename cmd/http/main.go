package main

import (
	"context"
	"fmt"
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
	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg := config.Load(os.Getenv)

	stopTracing, err := telemetry.SetupAll(ctx, "./logs/app.log", cfg.LogFormat)
	if err != nil {
		return fmt.Errorf("setup tracing: %w", err)
	}
	defer stopTracing()

	pool, err := postgres.OpenDB(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
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
	return nil
}
