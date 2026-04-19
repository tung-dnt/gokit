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

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	opentelemetry "github.com/xavidop/genkit-opentelemetry-go"
	"go.opentelemetry.io/otel"

	_ "gokit/docs"
	"gokit/internal/app"
	"gokit/internal/recipe"
	"gokit/internal/user"
	"gokit/pkg/config"
	router "gokit/pkg/http"
	"gokit/pkg/logger"
	"gokit/pkg/postgres"
	pgdb "gokit/pkg/postgres/db"
	"gokit/pkg/recovery"
	"gokit/pkg/telemetry"
	cv "gokit/pkg/validator"
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

	// Unified telemetry: traces + metrics + logs (all via OTLP to SigNoz).
	stopTelemetry, err := telemetry.SetupAll(ctx, cfg.LogFormat)
	if err != nil {
		return fmt.Errorf("setup telemetry: %w", err)
	}
	defer stopTelemetry()

	// Genkit OTEL plugin — exports Genkit-internal spans (LLM calls, embeddings,
	// retrieval, flows) via OTLP alongside the app's own spans.
	otelPlugin := opentelemetry.New(opentelemetry.Config{
		ServiceName:    "gokit",
		ForceExport:    true,
		OTLPEndpoint:   telemetry.EndpointHTTP(),
		OTLPUseHTTP:    true,
		MetricInterval: 15 * time.Second,
	})

	ai := genkit.Init(ctx,
		genkit.WithPlugins(&googlegenai.GoogleAI{}, otelPlugin),
		genkit.WithDefaultModel("googleai/gemini-2.5-flash"),
		genkit.WithPromptDir("./prompts"),
	)

	pool, err := postgres.OpenDB(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()

	a := &app.App{
		Queries:   pgdb.New(pool),
		Validator: cv.New(),
		Tracer:    otel.GetTracerProvider(),
		Agent:     ai,
	}

	r := router.NewRouter(router.WithInstrumentation("http.server"))
	r.Use(logger.Middleware)
	r.Use(recovery.Middleware)

	recipeMod, err := recipe.NewModule(a)
	if err != nil {
		return fmt.Errorf("recipe module: %w", err)
	}

	r.Group("/v1", func(g *router.Group) {
		g.Prefix("/api")
		g.ANY("/swagger/", httpSwagger.WrapHandler)
		g.Group("/users", user.NewModule(a).RegisterRoutes)
		g.Group("/agents", func(g *router.Group) {
			g.Group("/recipes", recipeMod.RegisterRoutes)
		})
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
