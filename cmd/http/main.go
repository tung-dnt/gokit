package main

import (
	"database/sql"
	"os"
	"time"

	_ "modernc.org/sqlite"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"

	_ "restful-boilerplate/dx/docs"

	"restful-boilerplate/biz/user"
	"restful-boilerplate/pkg/config"
	cv "restful-boilerplate/pkg/validator"
)

func registerRouters(g *echo.Group, db *sql.DB) {
	user.NewController(db).RegisterRoutes(g.Group("/users"))
	// add new domains: xxx.NewController(db).RegisterRoutes(g.Group("/xxx"))
}

//	@title          Restful Boilerplate API
//	@version        1.0
//	@description    Go RESTful API boilerplate built on Echo v5 + SQLite.
//	@host           localhost:8080
//	@BasePath       /api
//	@schemes        http
func main() {
	db, err := sql.Open("sqlite", "./data.db")
	if err != nil {
		panic("open db: " + err.Error())
	}
	defer db.Close()

	e := echo.New()
	e.Validator = cv.New()

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.ContextTimeout(5 * time.Second))
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 5}))
	e.Use(middleware.Secure())
	e.Use(middleware.CSRF())

	registerRouters(e.Group("/api"), db)
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	cfg := config.Load(os.Getenv).Server
	if err := e.Start(cfg.Host + ":" + cfg.Port); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}
