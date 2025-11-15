package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/iwnmname/PR-service.git/internal/config"
	"github.com/iwnmname/PR-service.git/internal/handler"
	"github.com/iwnmname/PR-service.git/internal/middleware"
	"github.com/iwnmname/PR-service.git/internal/pkg/transaction"
	prRepo "github.com/iwnmname/PR-service.git/internal/repository/pr"
	teamRepo "github.com/iwnmname/PR-service.git/internal/repository/team"
	userRepo "github.com/iwnmname/PR-service.git/internal/repository/user"
	prUsecase "github.com/iwnmname/PR-service.git/internal/usecase/pr"
	teamUsecase "github.com/iwnmname/PR-service.git/internal/usecase/team"
	userUsecase "github.com/iwnmname/PR-service.git/internal/usecase/user"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := sql.Open("postgres", cfg.Database.DSN())
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	log.Println("Connected to database")

	txManager := transaction.NewManager(db)

	userRepository := userRepo.NewRepository(db)
	teamRepository := teamRepo.NewRepository(db)
	prRepository := prRepo.NewRepository(db)

	teamUsecaseInstance := teamUsecase.NewUsecase(userRepository, teamRepository, txManager)
	userUsecaseInstance := userUsecase.NewUsecase(userRepository)
	prUsecaseInstance := prUsecase.NewUsecase(userRepository, prRepository, txManager)

	teamHandler := handler.NewTeamHandler(teamUsecaseInstance)
	userHandler := handler.NewUserHandler(userUsecaseInstance)
	prHandler := handler.NewPRHandler(prUsecaseInstance)

	mux := http.NewServeMux()

	mux.HandleFunc("/team/add", teamHandler.CreateTeam)
	mux.HandleFunc("/team/get", teamHandler.GetTeam)
	mux.HandleFunc("/users/setIsActive", userHandler.SetActive)
	mux.HandleFunc("/pullRequest/create", prHandler.CreatePR)
	mux.HandleFunc("/pullRequest/merge", prHandler.MergePR)
	mux.HandleFunc("/pullRequest/reassign", prHandler.ReassignReviewer)
	mux.HandleFunc("/users/getReview", prHandler.GetUserReviews)

	handler := middleware.Recovery(middleware.Logger(mux))

	server := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("Server starting on port %s", cfg.App.Port)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		log.Printf("Shutdown signal received: %v", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			server.Close()
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}

		log.Println("Server stopped gracefully")
	}

	return nil
}
