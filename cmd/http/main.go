package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"strings"
	"time"

	"restful-boilerplate/biz/user"
	_ "restful-boilerplate/dx/docs"
	"restful-boilerplate/pkg/config"
	"restful-boilerplate/pkg/metrics"
	"restful-boilerplate/pkg/middleware"
	"restful-boilerplate/pkg/otelecho"
	"restful-boilerplate/pkg/telemetry"
	cv "restful-boilerplate/pkg/validator"
	sqlitedb "restful-boilerplate/repo/sqlite"

	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
	_ "modernc.org/sqlite"
)

func registerRouters(g *echo.Group, db *sql.DB) {
	user.NewController(db).RegisterRoutes(g.Group("/users"))
	// add new domains: xxx.NewController(db).RegisterRoutes(g.Group("/xxx"))
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

	db, err := sqlitedb.OpenDB(ctx, "./data.db")
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
