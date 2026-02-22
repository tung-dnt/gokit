package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	"restful-boilerplate/biz/user"
	_ "restful-boilerplate/dx/docs"
	"restful-boilerplate/pkg/config"
	"restful-boilerplate/pkg/logger"
	"restful-boilerplate/pkg/metrics"
	"restful-boilerplate/pkg/middleware"
	"restful-boilerplate/pkg/otelecho"
	"restful-boilerplate/pkg/telemetry"
	cv "restful-boilerplate/pkg/validator"

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

	stopTracing, err := setupTracing(ctx)
	if err != nil {
		slog.Error("failed to setup tracing", "error", err)
		os.Exit(1)
	}
	defer stopTracing()

	db, err := openDB(ctx, "./data.db")
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
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
	e.Use(echomw.CSRF())

	registerRouters(e.Group("/api"), db)
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	cfg := config.Load(os.Getenv).Server
	if err := e.Start(cfg.Host + ":" + cfg.Port); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}

// setupTracing initialises OpenTelemetry tracing and structured log file output.
// Returns a single cleanup func that flushes spans and closes the log file.
func setupTracing(ctx context.Context) (func(), error) {
	shutdownTracer, err := telemetry.Setup(ctx)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll("./logs", 0o750); err != nil { //nolint:gosec // 0750 intentional
		_ = shutdownTracer(ctx)
		return nil, fmt.Errorf("create logs dir: %w", err)
	}

	closeLog, err := logger.Setup("./logs/app.log")
	if err != nil {
		_ = shutdownTracer(ctx)
		return nil, fmt.Errorf("setup logger: %w", err)
	}

	return func() {
		_ = closeLog()
		_ = shutdownTracer(ctx)
	}, nil
}

// openDB opens and configures a SQLite database at the given path.
// Single connection serialises access at the Go level, preventing SQLITE_BUSY
// between goroutines. busy_timeout is per-connection so must be set on every open.
func openDB(ctx context.Context, path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(ctx, `PRAGMA busy_timeout=5000;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("configure db: %w", err)
	}

	return db, nil
}
