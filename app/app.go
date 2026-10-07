package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"orders_backend/config"
	"orders_backend/database"
	"orders_backend/health"
	"orders_backend/router"
)

func Run(ctx context.Context) error {
	// 1. โหลด configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 2. เชื่อมต่อ MySQL
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	db, err := database.Open(connectCtx, cfg.DatabaseURL)
	cancel()

	if err != nil {
		return fmt.Errorf("connect MySQL: %w", err)
	}
	defer db.Close()

	log.Println("MySQL connected")

	healthHandler := health.NewHandler(db)

	// สร้าง Gin router
	httpHandler := router.New(router.Dependencies{
		Health: healthHandler,
	})

	// สร้าง HTTP server โดยใช้ Gin จัดการ request
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("API starting on %s using Gin", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	// รอ server error หรือสัญญาณให้หยุด
	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("HTTP server stopped: %w", err)

	case <-ctx.Done():
		log.Println("Shutting down API")

		// ใช้ context ใหม่ เพราะ ctx เดิมถูกยกเลิกแล้ว
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			8*time.Second,
		)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			// หากรอ request ไม่สำเร็จภายในเวลาที่กำหนด
			_ = server.Close()
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}

		log.Println("API shutdown complete")
		return nil
	}
}
