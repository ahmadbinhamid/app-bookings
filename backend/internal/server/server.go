package server

import (
	"database/sql"
	"net/http"
	"reflect"
	"strings"

	"app-booking/internal/config"
	"app-booking/internal/flowpos"
	"app-booking/internal/modules/assignments"
	"app-booking/internal/modules/booking"
	"app-booking/internal/modules/employee"
	"app-booking/internal/modules/installation"
	"app-booking/internal/modules/location"
	"app-booking/internal/modules/schedules"
	"app-booking/internal/modules/services"
	"app-booking/internal/modules/sync"
	"app-booking/internal/modules/timeoff"
	"app-booking/internal/server/handlers"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// Server wires the HTTP engine, routes, and dependencies together.
type Server struct {
	cfg    config.Config
	engine *gin.Engine

	// SyncScheduler is started by cmd/server/main.go (`go srv.SyncScheduler.Start(ctx)`)
	// after New() returns — kept separate from route wiring since it isn't
	// an HTTP concern.
	SyncScheduler *sync.Scheduler
}

// New builds the router and mounts every feature's routes.
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
	instRepo := installation.NewRepository(conn)
	instSvc := installation.NewService(instRepo)

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

	// --- Repositories + services for every domain module ---
	locRepo := location.NewRepository(conn)
	locSvc := location.NewService(locRepo)
	empRepo := employee.NewRepository(conn)

	// FlowPOS sync feature: pulls locations + employees per tenant. See
	// internal/modules/sync for the orchestration and internal/flowpos for
	// the API client — instRepo is reused directly here since sync needs the
	// raw api_key, which the installation *service* doesn't expose (its
	// Installation JSON marshals APIKey as "-"). Built here (rather than
	// further down, where it originally lived) so the lifecycle handler
	// below can trigger an immediate sync on install — a tenant that just
	// installed shouldn't have to wait up to an hour for the scheduler's
	// next tick, or need an admin to know a manual-trigger endpoint exists.
	flowposClient := flowpos.NewClient(cfg.FlowposAPIURL)
	syncSvc := sync.NewService(locRepo, empRepo, instRepo, flowposClient)

	// Marketplace lifecycle: FlowPOS calls these directly (no tenant JWT),
	// signing each request with this app's signing secret. Mounted at the
	// public root, not under /api, per the marketplace listing contract.
	lifecycle := handlers.NewLifecycleHandler(instSvc, syncSvc)
	verifySignature := handlers.SignatureMiddleware(cfg.SigningSecret)
	r.POST("/install", verifySignature, lifecycle.Install)
	r.POST("/uninstall", verifySignature, lifecycle.Uninstall)
	r.POST("/webhooks", verifySignature, lifecycle.Webhook)

	svcRepo := services.NewRepository(conn)
	svcSvc := services.NewService(svcRepo)
	assignRepo := assignments.NewRepository(conn)
	assignSvc := assignments.NewService(assignRepo)
	scheduleRepo := schedules.NewRepository(conn)
	scheduleSvc := schedules.NewService(scheduleRepo)
	timeOffRepo := timeoff.NewRepository(conn)
	timeOffSvc := timeoff.NewService(timeOffRepo)
	bookingRepo := booking.NewRepository(conn)
	bookingSvc := booking.NewService(bookingRepo, svcSvc, locSvc)
	// empSvc depends on bookingRepo (as employee.BookingConflictChecker) to
	// block AssignLocation from moving an employee off a location where they
	// still have future bookings — see employee.Service.AssignLocation.
	empSvc := employee.NewService(empRepo, bookingRepo)

	locationHandler := handlers.NewLocationHandler(locSvc)
	employeeHandler := handlers.NewEmployeeHandler(empSvc, locSvc)
	serviceHandler := handlers.NewServiceHandler(svcSvc)
	assignmentHandler := handlers.NewAssignmentHandler(assignSvc, empSvc)
	scheduleHandler := handlers.NewScheduleHandler(scheduleSvc)
	timeOffHandler := handlers.NewTimeOffHandler(timeOffSvc)
	bookingHandler := handlers.NewBookingHandler(bookingSvc)

	protected.GET("/locations", locationHandler.List)

	// Tenant-wide employee routes — NOT nested under /locations/:locationId,
	// since assigning an employee reaches across (or out of) a location, not
	// within one already-verified one. Each verifies tenant ownership itself
	// — see EmployeeHandler.AssignLocation's doc comment.
	protected.GET("/employees", employeeHandler.ListAll)
	protected.GET("/employees/unassigned", employeeHandler.ListUnassigned)
	protected.PATCH("/employees/:employeeId/location", employeeHandler.AssignLocation)

	// Every route below is nested under a specific, tenant-owned location —
	// RequireLocationOwnership (internal/server/handlers/ownership.go) is
	// the single shared check every one of them relies on, instead of each
	// handler re-implementing its own (that's what the Phase 2 employees
	// endpoint originally did ad hoc — extracted per this round's review).
	locGroup := protected.Group("/locations/:locationId")
	locGroup.Use(handlers.RequireLocationOwnership(locSvc))

	locGroup.PATCH("/timezone", locationHandler.SetTimezone)
	locGroup.GET("/employees", employeeHandler.ListByLocation)

	locGroup.GET("/services", serviceHandler.List)
	locGroup.POST("/services", serviceHandler.Create)

	// :serviceId routes additionally verify the service belongs to this
	// location (RequireServiceInLocation) before anything else runs.
	svcItemGroup := locGroup.Group("/services/:serviceId")
	svcItemGroup.Use(handlers.RequireServiceInLocation(svcSvc))
	svcItemGroup.GET("", serviceHandler.Get)
	svcItemGroup.PUT("", serviceHandler.Update)
	svcItemGroup.DELETE("", serviceHandler.Delete)
	svcItemGroup.GET("/employees", assignmentHandler.ListForService)
	svcItemGroup.POST("/employees", assignmentHandler.Assign)
	svcItemGroup.DELETE("/employees/:employeeId", assignmentHandler.Unassign)

	// :employeeId routes additionally verify the employee belongs to this
	// location (RequireEmployeeInLocation) before anything else runs.
	empItemGroup := locGroup.Group("/employees/:employeeId")
	empItemGroup.Use(handlers.RequireEmployeeInLocation(empSvc))
	empItemGroup.GET("/schedules", scheduleHandler.List)
	empItemGroup.POST("/schedules", scheduleHandler.Create)
	empItemGroup.DELETE("/schedules/:scheduleId", scheduleHandler.Delete)
	empItemGroup.GET("/time-off", timeOffHandler.List)
	empItemGroup.POST("/time-off", timeOffHandler.Create)
	empItemGroup.DELETE("/time-off/:timeOffId", timeOffHandler.Delete)

	// Bookings: propose/confirm are location-level (no booking exists yet);
	// everything else is nested under a specific, already-owned booking via
	// RequireBookingInLocation.
	locGroup.POST("/bookings/propose", bookingHandler.Propose)
	locGroup.POST("/bookings/confirm", bookingHandler.Confirm)
	locGroup.GET("/bookings", bookingHandler.List)

	bookingItemGroup := locGroup.Group("/bookings/:bookingId")
	bookingItemGroup.Use(handlers.RequireBookingInLocation(bookingSvc))
	bookingItemGroup.GET("", bookingHandler.Get)
	bookingItemGroup.POST("/cancel", bookingHandler.Cancel)
	bookingItemGroup.POST("/reschedule", bookingHandler.Reschedule)
	bookingItemGroup.POST("/segments/:segmentId/cancel", bookingHandler.CancelSegment)
	bookingItemGroup.PATCH("/segments/:segmentId/complete", bookingHandler.CompleteSegment)

	// syncSvc itself was built earlier (see above, near locRepo/empRepo) so
	// the lifecycle handler could use it too — just the route + scheduler
	// wiring is left to do here.
	protected.POST("/sync/trigger", handlers.NewSyncHandler(syncSvc).Trigger)
	syncScheduler := sync.NewScheduler(instRepo, syncSvc, cfg.SyncInterval)

	return &Server{cfg: cfg, engine: r, SyncScheduler: syncScheduler}
}

// Run starts listening on the configured port.
func (s *Server) Run() error {
	return s.engine.Run(":" + s.cfg.Port)
}

// Handler exposes the underlying http.Handler for in-process HTTP testing
// (httptest) without needing a real listening port — see
// internal/server/ownership_test.go.
func (s *Server) Handler() http.Handler {
	return s.engine
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
