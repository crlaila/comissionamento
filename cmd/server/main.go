package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"comissionamento/internal/handler"
	"comissionamento/internal/hinova"
	"comissionamento/internal/middleware"
	"comissionamento/internal/repository"
	"comissionamento/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		slog.Error("JWT_SECRET environment variable not set")
		os.Exit(1)
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		slog.Error("DATABASE_URL environment variable not set")
		os.Exit(1)
	}

	// Connect to database (use background context, not timeout)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	slog.Info("database connected successfully")

	// Initialize repositories and services
	slog.Info("creating repositories", "pool_not_nil", pool != nil)
	userRepo := repository.NewUserRepository(pool)
	slog.Info("user repository created")
	periodRepo := repository.NewPeriodRepository(pool)
	eventRepo := repository.NewMemberEventRepository(pool)
	goalRepo := repository.NewGoalRepository(pool)
	statementRepo := repository.NewStatementRepository(pool)
	authService := service.NewAuthService(jwtSecret, userRepo)
	authMiddleware := middleware.NewAuthMiddleware(authService)
	commissionService := service.NewCommissionService(goalRepo, periodRepo, userRepo, statementRepo, eventRepo, pool)

	// Initialize Hinova client
	hinovaToken := os.Getenv("HINOVA_API_TOKEN")
	var hinovaClient hinova.HinovaClient
	if hinovaToken != "" {
		hinovaClient = hinova.NewHTTPClient("https://api.hinova.com.br/api/sga/v2", hinovaToken)
	} else {
		slog.Warn("HINOVA_API_TOKEN not set, using mock client for development")
		hinovaClient = hinova.NewMockClient()
	}

	// Initialize Sync service with configurable interval
	syncInterval := 5 * time.Minute // default
	if syncIntervalStr := os.Getenv("SYNC_INTERVAL"); syncIntervalStr != "" {
		if minutes, err := strconv.Atoi(syncIntervalStr); err == nil && minutes > 0 {
			syncInterval = time.Duration(minutes) * time.Minute
		}
	}
	syncService := service.NewSyncService(hinovaClient, eventRepo, syncInterval)

	// Initialize repositories for handlers
	auditRepo := repository.NewAuditLogRepository(pool)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(authService, userRepo)
	periodHandler := handler.NewPeriodHandler(periodRepo)
	syncHandler := handler.NewSyncHandler(syncService)
	dashboardHandler := handler.NewDashboardHandler(commissionService)
	goalHandler := handler.NewGoalHandler(goalRepo, periodRepo)
	eventHandler := handler.NewEventHandler(eventRepo, userRepo, periodRepo)
	statementHandler := handler.NewStatementHandler(statementRepo, commissionService, auditRepo, periodRepo, userRepo, pool)
	reportHandler := handler.NewReportHandler(statementRepo, periodRepo, userRepo, goalRepo, eventRepo)

	mux := http.NewServeMux()

	// Health endpoint (no auth required)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","timestamp":"%s"}`, time.Now().UTC().Format(time.RFC3339))
	})

	// Auth endpoints (no auth required)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.Handle("POST /api/auth/refresh", authMiddleware.Authenticate(http.HandlerFunc(authHandler.Refresh)))
	mux.HandleFunc("POST /api/auth/logout", authHandler.Logout)

	// User endpoints (admin only)
	mux.Handle("GET /api/users", authMiddleware.Authenticate(authMiddleware.RequireRole("admin")(http.HandlerFunc(userHandler.List))))
	mux.Handle("POST /api/users", authMiddleware.Authenticate(authMiddleware.RequireRole("admin")(http.HandlerFunc(userHandler.Create))))
	mux.Handle("PUT /api/users/{id}", authMiddleware.Authenticate(authMiddleware.RequireRole("admin")(http.HandlerFunc(userHandler.Update))))

	// Period endpoints (admin only)
	mux.Handle("GET /api/periods", authMiddleware.Authenticate(authMiddleware.RequireRole("admin")(http.HandlerFunc(periodHandler.List))))
	mux.Handle("POST /api/periods", authMiddleware.Authenticate(authMiddleware.RequireRole("admin")(http.HandlerFunc(periodHandler.Create))))
	mux.Handle("PUT /api/periods/{id}", authMiddleware.Authenticate(authMiddleware.RequireRole("admin")(http.HandlerFunc(periodHandler.Update))))

	// Sync endpoints
	mux.HandleFunc("GET /api/sync/status", syncHandler.GetStatus)
	mux.Handle("POST /api/sync/trigger", authMiddleware.Authenticate(authMiddleware.RequireRole("admin")(http.HandlerFunc(syncHandler.Trigger))))

	// Dashboard endpoints
	mux.Handle("GET /api/dashboard/rep", authMiddleware.Authenticate(http.HandlerFunc(dashboardHandler.GetRepDashboard)))
	mux.Handle("GET /api/dashboard/team", authMiddleware.Authenticate(http.HandlerFunc(dashboardHandler.GetTeamDashboard)))
	mux.Handle("GET /api/dashboard/org", authMiddleware.Authenticate(http.HandlerFunc(dashboardHandler.GetOrgDashboard)))

	// Goal endpoints
	mux.Handle("GET /api/goals", authMiddleware.Authenticate(http.HandlerFunc(goalHandler.ListGoals)))
	mux.Handle("POST /api/goals", authMiddleware.Authenticate(http.HandlerFunc(goalHandler.CreateGoal)))
	mux.Handle("PUT /api/goals/{id}", authMiddleware.Authenticate(http.HandlerFunc(goalHandler.UpdateGoal)))

	// Event endpoints
	mux.Handle("GET /api/events", authMiddleware.Authenticate(http.HandlerFunc(eventHandler.ListEvents)))
	mux.Handle("GET /api/events/{id}", authMiddleware.Authenticate(http.HandlerFunc(eventHandler.GetEvent)))

	// Statement endpoints
	mux.Handle("GET /api/statements", authMiddleware.Authenticate(http.HandlerFunc(statementHandler.GetStatements)))
	mux.Handle("GET /api/statements/{id}", authMiddleware.Authenticate(http.HandlerFunc(statementHandler.GetStatement)))
	mux.Handle("POST /api/statements/generate", authMiddleware.Authenticate(http.HandlerFunc(statementHandler.GenerateStatements)))
	mux.Handle("POST /api/statements/{id}/approve", authMiddleware.Authenticate(http.HandlerFunc(statementHandler.ApproveStatement)))
	mux.Handle("POST /api/statements/{id}/reject", authMiddleware.Authenticate(http.HandlerFunc(statementHandler.RejectStatement)))

	// Report endpoints
	mux.Handle("GET /api/reports/commission-detail", authMiddleware.Authenticate(http.HandlerFunc(reportHandler.GetCommissionDetail)))
	mux.Handle("GET /api/reports/team-summary", authMiddleware.Authenticate(http.HandlerFunc(reportHandler.GetTeamSummary)))
	mux.Handle("GET /api/reports/liability", authMiddleware.Authenticate(http.HandlerFunc(reportHandler.GetLiability)))
	mux.Handle("GET /api/reports/export", authMiddleware.Authenticate(http.HandlerFunc(reportHandler.GetExport)))

	// Create a context for the sync worker that can be cancelled
	syncCtx, syncCancel := context.WithCancel(context.Background())
	defer syncCancel()

	// Start sync worker in background
	go syncService.StartPolling(syncCtx)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigChan
		slog.Info("shutdown signal received", "signal", sig.String())

		// Cancel sync worker context
		syncCancel()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
	}()

	slog.Info("server starting", "port", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}
