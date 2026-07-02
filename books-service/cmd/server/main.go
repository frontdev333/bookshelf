package main

import (
	"bookshelf/books-service/internal/client"
	"bookshelf/books-service/internal/config"
	"bookshelf/books-service/internal/handler"
	middleware2 "bookshelf/books-service/internal/middleware"
	"bookshelf/books-service/internal/repository"
	"bookshelf/books-service/internal/service"
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
	_ "github.com/lib/pq"
)

const stdTimeout = 30 * time.Second

func main() {
	cfg := config.Load()

	port := cfg.Port

	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to db error: %s", err.Error())
	}
	defer db.Close()

	slog.Info("connected to database")
	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(3)
	db.SetConnMaxLifetime(3 * time.Minute)

	bookRepo := repository.NewBookRepository(db)
	reviewRepo := repository.NewReviewRepository(db)

	bookService := service.NewBookService(bookRepo)
	reviewService := service.NewReviewService(reviewRepo)

	bookHandler := handler.NewBookHandler(bookService)
	reviewHandler := handler.NewReviewHandler(reviewService)

	authClient := client.NewAuthClient(cfg.AuthServiceURL, 15*time.Second)

	mux := chi.NewRouter()
	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
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

	mux.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", bookHandler.Health)
		r.Get("/ready", bookHandler.Ready)
		r.Get("/books", bookHandler.List)
		r.Get("/books/{book_id}", bookHandler.GetByID)
		r.Get("/books/{book_id}/reviews", reviewHandler.List)
		r.Get("/reviews/{reviewId}", reviewHandler.GetReview)

		r.Group(func(r chi.Router) {
			r.Use(middleware2.AuthMiddleware(authClient))

			r.Post("/books", bookHandler.Create)
			r.Put("/books/{book_id}", bookHandler.Update)
			r.Delete("/books/{book_id}", bookHandler.Delete)
			r.Post("/books/{book_id}/reviews", reviewHandler.Create)
			r.Put("/reviews/{reviewId}", reviewHandler.Update)
			r.Delete("/reviews/{reviewId}", reviewHandler.Delete)
		})
	})

	slog.Info("Server starting", "port", port)

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
