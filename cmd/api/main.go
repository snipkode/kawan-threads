package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kawan-threads/internal/application/runtimeconfig"
	"kawan-threads/internal/application/usecase"
	"kawan-threads/internal/config"
	"kawan-threads/internal/domain/port"
	"kawan-threads/internal/domain/repository"
	"kawan-threads/internal/infrastructure/filestore"
	"kawan-threads/internal/infrastructure/firebase"
	"kawan-threads/internal/infrastructure/gemini"
	"kawan-threads/internal/infrastructure/threads"
	"kawan-threads/internal/interfaces/http/handler"
	"kawan-threads/internal/interfaces/http/middleware"
	"kawan-threads/internal/interfaces/http/router"
	applogger "kawan-threads/internal/logger"
)

func main() {
	// -------------------------------------------------------------------------
	// Configuration
	// -------------------------------------------------------------------------
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// -------------------------------------------------------------------------
	// Logger
	// -------------------------------------------------------------------------
	level := "info"
	if cfg.App.Env == "development" {
		level = "debug"
	}
	applogger.Init(level)
	log := applogger.Global
	log.Info("starting API server", "env", cfg.App.Env, "port", cfg.App.Port)

	// -------------------------------------------------------------------------
	// Datastore — "file" (dev/demo, no credentials) or "firebase" (default)
	// -------------------------------------------------------------------------
	var contentRepo repository.ContentRepository
	var contentVersionRepo repository.ContentVersionRepository
	var queueRepo repository.QueueRepository
	var scheduleRepo repository.ScheduleRepository
	var publishedPostRepo repository.PublishedPostRepository
	var performanceRepo repository.PostPerformanceRepository
	var topicRepo repository.TopicRepository
	var historyRepo repository.HistoryRepository
	var experimentRepo repository.ExperimentRepository
	var settingsRepo repository.SettingsRepository

	switch cfg.Store.DataStore {
	case "file":
		fs, err := filestore.Open(cfg.Store.DataFile)
		if err != nil {
			log.Error("failed to open file datastore", "error", err)
			os.Exit(1)
		}
		log.Info("using file datastore", "path", cfg.Store.DataFile)
		contentRepo = fs.Content
		contentVersionRepo = fs.ContentVersion
		queueRepo = fs.Queue
		scheduleRepo = fs.Schedule
		publishedPostRepo = fs.PublishedPost
		performanceRepo = fs.Performance
		topicRepo = fs.Topic
		historyRepo = fs.History
		experimentRepo = fs.Experiment
		settingsRepo = fs.Settings

	default: // firebase
		firebaseClient, err := firebase.NewFirebaseClient(cfg)
		if err != nil {
			log.Error("failed to initialise Firebase client", "error", err)
			os.Exit(1)
		}
		contentRepo = firebase.NewContentRepository(firebaseClient)
		contentVersionRepo = firebase.NewContentVersionRepository(firebaseClient)
		queueRepo = firebase.NewQueueRepository(firebaseClient)
		scheduleRepo = firebase.NewScheduleRepository(firebaseClient)
		publishedPostRepo = firebase.NewPublishedPostRepository(firebaseClient)
		performanceRepo = firebase.NewPostPerformanceRepository(firebaseClient)
		topicRepo = firebase.NewTopicRepository(firebaseClient)
		historyRepo = firebase.NewHistoryRepository(firebaseClient)
		experimentRepo = firebase.NewExperimentRepository(firebaseClient)
		settingsRepo = firebase.NewSettingsRepository(firebaseClient)
	}

	_ = publishedPostRepo // kept for parity with the publishing worker surface
	_ = experimentRepo    // reserved for A/B testing support

	// -------------------------------------------------------------------------
	// Runtime configuration — most settings are UI-editable via /api/settings
	// and shared with the worker through the datastore. Only the datastore and
	// Firebase service account remain environment-only.
	// -------------------------------------------------------------------------
	runtimeCfg := runtimeconfig.New(settingsRepo, cfg, log)
	runtimeCfg.Load(context.Background())

	aiConfigured := runtimeCfg.GeminiConfigured()
	threadsConfigured := runtimeCfg.ThreadsConfigured()
	threadsConnected := runtimeCfg.ThreadsConnected()

	// -------------------------------------------------------------------------
	// AI Provider (Gemini) — reads credentials from the runtime settings, so
	// it can be (re)configured through the UI without a restart.
	// -------------------------------------------------------------------------
	aiProvider := gemini.NewGeminiAdapter(runtimeCfg, log)
	if aiConfigured {
		log.Info("gemini adapter initialised", "model", runtimeCfg.Str("gemini_model", ""), "managed_via_ui", true)
	} else {
		log.Info("Gemini has no API key — AI generation unavailable until set in Settings")
	}

	// -------------------------------------------------------------------------
	// Threads Adapter — always constructed; credentials come from runtime
	// settings. Publishing/analytics workers self-gate on configuration.
	// -------------------------------------------------------------------------
	threadsAdapter := threads.NewThreadsAdapter(runtimeCfg, log)
	if threadsConfigured {
		log.Info("threads adapter configured",
			"user_id", runtimeCfg.Str("threads_user_id", ""),
			"connected", threadsConnected,
			"managed_via_ui", true,
		)
	} else {
		log.Info("Threads has no client credentials — set them in Settings to connect")
	}

	// -------------------------------------------------------------------------
	// Use Cases
	// -------------------------------------------------------------------------
	contentUC := &usecase.ContentUseCase{
		ContentRepo: contentRepo,
		VersionRepo: contentVersionRepo,
		HistoryRepo: historyRepo,
		AIProvider:  aiProvider,
		Logger:      log,
	}

	approvalUC := &usecase.ApprovalUseCase{
		ContentRepo: contentRepo,
		VersionRepo: contentVersionRepo,
		QueueRepo:   queueRepo,
		HistoryRepo: historyRepo,
		Logger:      log,
	}

	// -------------------------------------------------------------------------
	// Handlers
	// -------------------------------------------------------------------------
	base := handler.NewBaseHandler(log, contentRepo, queueRepo, scheduleRepo, performanceRepo, topicRepo, historyRepo)

	var threadsPort port.ThreadsPort
	if threadsAdapter != nil {
		threadsPort = threadsAdapter
	}

	deps := router.Deps{
		Health:    handler.NewHealthHandler(),
		Dashboard: handler.NewDashboardHandler(base),
		Content:   handler.NewContentHandler(base, contentUC, approvalUC),
		Queue:     handler.NewQueueHandler(base, approvalUC),
		Schedule:  handler.NewScheduleHandler(base),
		Analytics: handler.NewAnalyticsHandler(base),
		Topic:     handler.NewTopicHandler(base),
		Settings:  handler.NewSettingsHandler(base, runtimeCfg),
		Auth:      handler.NewAuthHandler(threadsPort),
	}

	corsOrigins := []string{"*"}
	if cfg.Frontend.ViteAPIBaseURL != "" {
		corsOrigins = []string{cfg.Frontend.ViteAPIBaseURL}
	}

	// -------------------------------------------------------------------------
	// Router
	// -------------------------------------------------------------------------
	mux := router.New(deps, corsOrigins...)

	// Recovery wraps the mux; request-id + CORS handled inside router.
	var h http.Handler = mux
	h = middleware.Recovery(log)(h)

	// -------------------------------------------------------------------------
	// HTTP Server
	// -------------------------------------------------------------------------
	addr := fmt.Sprintf(":%s", cfg.App.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      h,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("HTTP server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// -------------------------------------------------------------------------
	// Graceful shutdown
	// -------------------------------------------------------------------------
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Info("shutdown signal received", "signal", sig.String())
	case err := <-serverErr:
		log.Error("server error", "error", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Info("shutting down HTTP server gracefully")
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server shutdown error", "error", err)
		os.Exit(1)
	}
	log.Info("server stopped cleanly")
}
