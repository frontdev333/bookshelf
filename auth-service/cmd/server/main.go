package main

import (
	"bookshelf/auth-svc/internal/config"
	"bookshelf/auth-svc/internal/handler"
	"bookshelf/auth-svc/internal/repository"
	"bookshelf/auth-svc/internal/service"
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jmoiron/sqlx"
)

const stdTimeout = 30 * time.Second

func main() {
	cfg := config.Load()
	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to db error: %s", err.Error())
	}
	defer db.Close()

	repo := repository.NewUserRepository(db)
	userService := service.NewUserService(repo, cfg.JWTSecret)
	h := handler.New(userService, cfg.JWTSecret)
	internalHandler := handler.NewInternalHandler(userService)

	mux := chi.NewRouter()
	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5174"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.Timeout(stdTimeout))

	mux.Get("/ready", h.Ready)
	mux.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", h.Health)

		r.Post("/auth/register", h.Register)
		r.Post("/auth/login", h.Login)

		r.Group(func(r chi.Router) {
			r.Use(handler.AuthMiddleware(userService))

			r.Post("/users/me", h.GetMe)
			r.Put("/users/me", h.UpdateMe)
		})
	})

	mux.Route("/internal/v1", func(r chi.Router) {
		r.Use(handler.ServiceKeyMiddleware(cfg.ServiceKey))
		r.Post("/auth/verify", internalHandler.VerifyToken)
		r.Post("/users/batch", internalHandler.GetUsersByIDs)
	})

	slog.Info("auth-service started", "port", cfg.Port)
	finish := make(chan os.Signal, 1)
	server := http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadTimeout:       stdTimeout,
		ReadHeaderTimeout: stdTimeout,
		WriteTimeout:      stdTimeout,
		IdleTimeout:       stdTimeout,
	}

	go func() {
		if err = server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server", "error", err)
			os.Exit(1)
		}
	}()

	signal.Notify(finish, os.Interrupt, syscall.SIGTERM)
	<-finish

	ctx, cancel := context.WithTimeout(context.Background(), stdTimeout)
	defer cancel()
	server.Shutdown(ctx)
}
