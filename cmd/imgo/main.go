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
	app.StartOverviewMetrics()
	httpServer := &http.Server{Addr: cfg.Addr, Handler: app.Router(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 90 * time.Second, WriteTimeout: 90 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 1 << 20}
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
