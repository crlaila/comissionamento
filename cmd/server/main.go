package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"comissionamento/internal/handler"
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

	// Connect to database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Initialize repositories and services
	userRepo := repository.NewUserRepository(pool)
	authService := service.NewAuthService(jwtSecret, userRepo)
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(authService, userRepo)

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
