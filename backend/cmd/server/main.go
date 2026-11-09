package main

import (
	"context"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"hooklab/internal/app"
	"hooklab/migrations"
	"log/slog"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--healthcheck" {
		client := &http.Client{Timeout: 2 * time.Second}
		res, err := client.Get("http://127.0.0.1:8080/readyz")
		if err != nil {
			os.Exit(1)
		}
		res.Body.Close()
		if res.StatusCode != 200 {
			os.Exit(1)
		}
		return
	}
	var proxies []string
	if raw := os.Getenv("TRUSTED_PROXIES"); raw != "" {
		for _, value := range strings.Split(raw, ",") {
			value = strings.TrimSpace(value)
			if _, err := netip.ParsePrefix(value); err != nil {
				if _, err = netip.ParseAddr(value); err != nil {
					slog.Error("invalid TRUSTED_PROXIES")
					os.Exit(1)
				}
			}
			proxies = append(proxies, value)
		}
	}

	dsn := os.Getenv("DATABASE_DSN")
	origin := os.Getenv("APP_ORIGIN")
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.Path != "" {
		slog.Error("APP_ORIGIN must be a single http(s) origin without trailing slash")
		os.Exit(1)
	}
	secure := os.Getenv("COOKIE_SECURE") != "false"
	if !secure && u.Scheme != "http" {
		slog.Error("use secure cookies for HTTPS")
		os.Exit(1)
	}
	maxBody := int64(1048576)
	if s := os.Getenv("MAX_BODY_BYTES"); s != "" {
		maxBody, err = strconv.ParseInt(s, 10, 64)
		if err != nil || maxBody < 1 || maxBody > 8*1048576 {
			slog.Error("MAX_BODY_BYTES must be 1..8388608")
			os.Exit(1)
		}
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		slog.Error("invalid database configuration")
		os.Exit(1)
	}
	defer db.Close()
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(3 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err = migrations.Run(ctx, db); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
	srv := &http.Server{Addr: ":8080", Handler: app.New(db, app.Config{Origin: origin, Secure: secure, MaxBody: maxBody, TrustedProxies: proxies, FrontendURL: os.Getenv("FRONTEND_URL")}), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32768}
	go func() {
		slog.Info("HookLab listening", "port", 8080)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdown, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()
	srv.Shutdown(shutdown)
}
