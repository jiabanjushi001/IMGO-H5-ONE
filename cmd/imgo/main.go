package main

import (
	"context"
	"flag"
	"imgo/internal/server"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	httpReadHeaderTimeout = 10 * time.Second
	httpRequestTimeout    = 30 * time.Minute
	httpIdleTimeout       = 90 * time.Second
)

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: httpReadHeaderTimeout,
		// A permitted 200 MB video can take well over 90 seconds on a normal
		// uplink. The request body remains capped by Router's MaxBytesReader.
		ReadTimeout:    httpRequestTimeout,
		WriteTimeout:   httpRequestTimeout,
		IdleTimeout:    httpIdleTimeout,
		MaxHeaderBytes: 1 << 20,
	}
}

func main() {
	migrate := flag.Bool("migrate", false, "apply additive migration to existing database")
	initialize := flag.Bool("init", false, "initialize an empty database")
	admin := flag.Bool("create-admin", false, "create admin using ADMIN_ACCOUNT and ADMIN_PASSWORD environment variables")
	flag.Parse()
	cfg, e := server.EnvConfig()
	if e != nil {
		slog.Error("configuration", "error", e)
		os.Exit(1)
	}
	app, e := server.New(cfg)
	if e != nil {
		slog.Error("startup", "error", e)
		os.Exit(1)
	}
	defer app.Close()
	if *migrate || *initialize || *admin {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if *migrate || *initialize {
			e = app.Migrate(ctx, *initialize)
		}
		if e == nil && *admin {
			e = app.CreateAdmin(ctx, os.Getenv("ADMIN_ACCOUNT"), os.Getenv("ADMIN_PASSWORD"))
		}
		if e != nil {
			slog.Error("database operation", "error", e)
			os.Exit(1)
		}
		slog.Info("database operation completed")
		return
	}
	upgradeCtx, upgradeCancel := context.WithTimeout(context.Background(), 10*time.Minute)
	e = app.EnsureAddonSchema(upgradeCtx)
	upgradeCancel()
	if e != nil {
		slog.Error("automatic database upgrade failed", "error", e)
		os.Exit(1)
	}
	if e = app.CheckSchema(context.Background()); e != nil {
		slog.Error("schema", "error", e)
		os.Exit(1)
	}
	app.StartBatchUserTasks()
	app.StartOverviewMetrics()
	httpServer := newHTTPServer(cfg.Addr, app.Router())
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		slog.Info("Imgo listening", "address", cfg.Addr)
		if e := httpServer.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			slog.Error("serve", "error", e)
			stop()
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if e = httpServer.Shutdown(shutdown); e != nil {
		slog.Error("shutdown", "error", e)
	}
}
