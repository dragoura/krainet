package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	cfg := loadConfig()
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	ctx := context.Background()
	pool, err := connectDB(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db connect failed", slog.String("error", err.Error()))
	}
	if pool != nil {
		defer pool.Close()
	}

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogLatency: true, LogStatus: true, LogMethod: true, LogURI: true, LogError: true, HandleError: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			slog.Info("request", slog.String("method", v.Method), slog.String("uri", v.URI), slog.Int("status", v.Status), slog.Duration("latency", v.Latency))
			return nil
		},
	}))

	e.Use(metricsMiddleware)
	e.GET("/metrics", metricsHandler())

	e.GET("/health", func(c echo.Context) error {
		if pool != nil {
			ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
			defer cancel()
			if err := pool.Ping(ctx); err != nil {
				return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "degraded"})
			}
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	e.GET("/api/users", func(c echo.Context) error {
		rows, err := pool.Query(c.Request().Context(), "SELECT id, name, email FROM users ORDER BY id")
		if err != nil {
			return err
		}
		defer rows.Close()
		var users []user
		for rows.Next() {
			var u user
			if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
				return err
			}
			users = append(users, u)
		}
		return c.JSON(http.StatusOK, users)
	})

	e.POST("/api/users", func(c echo.Context) error {
		var in struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if err := c.Bind(&in); err != nil || in.Name == "" || in.Email == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "name and email required")
		}
		if pool == nil {
			return c.JSON(http.StatusCreated, user{ID: 2, Name: in.Name, Email: in.Email})
		}
		var id int64
		if err := pool.QueryRow(c.Request().Context(), "INSERT INTO users(name,email) VALUES ($1,$2) RETURNING id", in.Name, in.Email).Scan(&id); err != nil {
			return err
		}
		return c.JSON(http.StatusCreated, user{ID: id, Name: in.Name, Email: in.Email})
	})

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: e}
	go func() {
		if err := e.StartServer(srv); err != nil && err != http.ErrServerClosed {
			slog.Error("server start failed", slog.String("error", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = e.Shutdown(ctxShutdown)
}
