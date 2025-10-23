package main

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requests = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "http_requests_total", Help: "Total HTTP requests"}, []string{"method", "path", "status"})
	latency  = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "http_request_duration_seconds", Help: "HTTP request latency"}, []string{"method", "path", "status"})
)

func init() {
	prometheus.MustRegister(requests, latency)
}

func metricsMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		path := c.Path()
		method := c.Request().Method
		err := next(c)
		status := c.Response().Status
		dur := time.Since(start).Seconds()
		code := strconv.Itoa(status)
		requests.WithLabelValues(method, path, code).Inc()
		latency.WithLabelValues(method, path, code).Observe(dur)
		return err
	}
}

func metricsHandler() echo.HandlerFunc {
	return echo.WrapHandler(promhttp.Handler())
}
