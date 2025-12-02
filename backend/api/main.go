package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/DeafMist/hot-tour-radar/backend/internal/config"
	"github.com/DeafMist/hot-tour-radar/backend/internal/elasticsearch"
	"github.com/DeafMist/hot-tour-radar/backend/internal/logger"
)

func main() {
	log := logger.New("api")
	cfg, err := config.LoadAPI()
	if err != nil {
		log.Error("load config", slog.Any("err", err))
		os.Exit(1)
	}

	esClient, err := elasticsearch.New(cfg.ElasticsearchAddr, cfg.ElasticsearchIndex, log)
	if err != nil {
		log.Error("init elasticsearch", slog.Any("err", err))
		os.Exit(1)
	}

	srv := &server{log: log, cfg: cfg, es: esClient}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/health", srv.handleHealth)
	r.Get("/news", srv.handleSearch)

	httpServer := &http.Server{
		Addr:              cfg.BindAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		log.Info("api server starting", slog.String("addr", cfg.BindAddr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server stopped", slog.Any("err", err))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown", slog.Any("err", err))
	}
}

type server struct {
	log *slog.Logger
	cfg *config.API
	es  *elasticsearch.Client
}

type errorResponse struct {
	Error string `json:"error"`
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.es.Health(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) handleSearch(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	keywords := parseCSV(r.URL.Query().Get("keywords"))
	source := strings.TrimSpace(r.URL.Query().Get("source"))

	from := clampInt(r.URL.Query().Get("from"), 0, 10_000)
	size := clampInt(r.URL.Query().Get("size"), s.cfg.DefaultPage, s.cfg.MaxPage)
	sort := strings.TrimSpace(r.URL.Query().Get("sort"))

	start := parseTime(r.URL.Query().Get("start"))
	end := parseTime(r.URL.Query().Get("end"))

	params := elasticsearch.SearchParams{
		Query:    query,
		Keywords: keywords,
		Source:   source,
		From:     from,
		Size:     size,
		Sort:     sort,
	}
	if start != nil {
		params.Start = start
	}
	if end != nil {
		params.End = end
	}

	result, err := s.es.SearchNews(ctx, params)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func parseTime(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if ts, err := time.Parse(time.RFC3339, raw); err == nil {
		return &ts
	}
	return nil
}

func parseCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func clampInt(raw string, fallback, max int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// nothing better to do
	}
}
