package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	_ "modernc.org/sqlite"

	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"

	"restful-boilerplate/app/user"
	_ "restful-boilerplate/docs"
	"restful-boilerplate/infra/config"
	"restful-boilerplate/infra/http/userhdl"
	"restful-boilerplate/infra/metrics"
	"restful-boilerplate/infra/middleware"
	"restful-boilerplate/infra/otelecho"
	infradb "restful-boilerplate/infra/sqlite"
	"restful-boilerplate/infra/sqlite/userrepo"
	"restful-boilerplate/infra/telemetry"
	cv "restful-boilerplate/infra/validator"
)

func registerRouters(g *echo.Group, db *sql.DB) {
	userRepo := userrepo.NewSQLite(db)
	userSvc := user.NewService(userRepo, otel.Tracer("user"))
	userhdl.NewHandler(userSvc).RegisterRoutes(g.Group("/users"))
	// add new domains: follow the same pattern above
}

// @title          Restful Boilerplate API
// @version        1.0
// @description    Go RESTful API boilerplate built on Echo v5 + SQLite.
// @host           localhost:8080
// @BasePath       /api
// @schemes        http
func main() {
	ctx := context.Background()

	stopTracing, err := telemetry.SetupAll(ctx, "./logs/app.log")
	if err != nil {
		slog.Error("failed to setup tracing", "error", err)
		os.Exit(1)
	}

	db, err := infradb.OpenDB(ctx, "./data.db")
	if err != nil {
		slog.Error("failed to open database", "error", err)
		stopTracing()
		os.Exit(1)
	}

	// All early-exit paths done; defers are safe from here.
	defer stopTracing()
	defer db.Close() //nolint:errcheck // best-effort close on exit

	e := echo.New()
	e.Validator = cv.New()
	metrics.Register(e)

	e.Use(otelecho.Middleware("restful-boilerplate"))
	e.Use(middleware.RequestLog)
	e.Use(echomw.Recover())
	e.Use(echomw.ContextTimeout(5 * time.Second))
	e.Use(echomw.GzipWithConfig(echomw.GzipConfig{Level: 5}))
	e.Use(echomw.Secure())
	e.Use(echomw.CSRFWithConfig(echomw.CSRFConfig{
		Skipper: func(c *echo.Context) bool {
			return strings.HasPrefix((*c).Request().URL.Path, "/swagger")
		},
	}))

	registerRouters(e.Group("/api"), db)
	e.GET("/swagger/*", echoSwagger.WrapHandler)
	e.GET("/swagger", func(c *echo.Context) error {
		return (*c).Redirect(301, "/swagger/index.html")
	})

	cfg := config.Load(os.Getenv).Server
	if err = e.Start(cfg.Host + ":" + cfg.Port); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}
