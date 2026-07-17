package server

import (
	"database/sql"
	"reflect"
	"strings"

	"app-booking/internal/config"
	"app-booking/internal/modules/installation"
	"app-booking/internal/server/handlers"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// Server wires the HTTP engine, routes, and dependencies together.
type Server struct {
	cfg    config.Config
	engine *gin.Engine
}

// New builds the router and mounts every feature's routes. This is boilerplate:
// only the installation/auth/health scaffolding exists so far — add new
// feature modules the same way the appointments/quotes apps do (a
// internal/modules/<feature> package + a handlers.New<Feature>Handler(...)
// call here).
func New(cfg config.Config, conn *sql.DB) *Server {
	useJSONFieldNames()

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		if err := conn.Ping(); err != nil {
			c.JSON(503, gin.H{"status": "error", "db": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	// installation feature (marketplace lifecycle)
	instSvc := installation.NewService(installation.NewRepository(conn))

	// Marketplace lifecycle: FlowPOS calls these directly (no tenant JWT),
	// signing each request with this app's signing secret. Mounted at the
	// public root, not under /api, per the marketplace listing contract.
	lifecycle := handlers.NewLifecycleHandler(instSvc)
	verifySignature := handlers.SignatureMiddleware(cfg.SigningSecret)
	r.POST("/install", verifySignature, lifecycle.Install)
	r.POST("/uninstall", verifySignature, lifecycle.Uninstall)
	r.POST("/webhooks", verifySignature, lifecycle.Webhook)

	api := r.Group("/api/v1")

	// Dev-only: mint a JWT for local testing (the real token is delivered by
	// the tenant dashboard). Disabled unless JWT_DEV_TOKENS=true.
	if cfg.AllowDevTokens {
		api.POST("/dev/token", handlers.NewDevTokenHandler(cfg.JWTSecret).Mint)
	}

	// Identity + installation state, protected by the tenant JWT.
	protected := api.Group("")
	protected.Use(handlers.AuthMiddleware(cfg.JWTSecret))
	protected.GET("/me", handlers.NewMeHandler(instSvc).Me)

	// Add new feature modules below, e.g.:
	//   fooSvc := foo.NewService(foo.NewRepository(conn))
	//   handlers.NewFooHandler(fooSvc).Register(api)

	return &Server{cfg: cfg, engine: r}
}

// Run starts listening on the configured port.
func (s *Server) Run() error {
	return s.engine.Run(":" + s.cfg.Port)
}

// useJSONFieldNames makes the binding validator report a field's JSON name
// (e.g. "name") instead of its Go struct name (e.g. "Name") in errors.
func useJSONFieldNames() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}
