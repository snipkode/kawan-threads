package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"kawan-threads/internal/application/runtimeconfig"
	"kawan-threads/internal/config"
	"kawan-threads/internal/domain/repository"
	"kawan-threads/internal/infrastructure/filestore"
	"kawan-threads/internal/infrastructure/firebase"
	"kawan-threads/internal/infrastructure/scheduler"
	"kawan-threads/internal/infrastructure/threads"
	applogger "kawan-threads/internal/logger"
	"kawan-threads/internal/worker/analytics"
	"kawan-threads/internal/worker/publishing"
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

	applogger.LogEvent(log, applogger.EventWorkerStarted,
		"env", cfg.App.Env,
		"timezone", cfg.Scheduler.Timezone,
	)

	// -------------------------------------------------------------------------
	// Datastore — "file" (dev/demo, no credentials) or "firebase" (default)
	// -------------------------------------------------------------------------
	var contentRepo repository.ContentRepository
	var queueRepo repository.QueueRepository
	var scheduleRepo repository.ScheduleRepository
	var publishedPostRepo repository.PublishedPostRepository
	var performanceRepo repository.PostPerformanceRepository
	var historyRepo repository.HistoryRepository
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
		queueRepo = fs.Queue
		scheduleRepo = fs.Schedule
		publishedPostRepo = fs.PublishedPost
		performanceRepo = fs.Performance
		historyRepo = fs.History
		settingsRepo = fs.Settings

	default: // firebase
		firebaseClient, err := firebase.NewFirebaseClient(cfg)
		if err != nil {
			log.Error("failed to initialise Firebase client", "error", err)
			os.Exit(1)
		}
		contentRepo = firebase.NewContentRepository(firebaseClient)
		queueRepo = firebase.NewQueueRepository(firebaseClient)
		scheduleRepo = firebase.NewScheduleRepository(firebaseClient)
		publishedPostRepo = firebase.NewPublishedPostRepository(firebaseClient)
		performanceRepo = firebase.NewPostPerformanceRepository(firebaseClient)
		historyRepo = firebase.NewHistoryRepository(firebaseClient)
		settingsRepo = firebase.NewSettingsRepository(firebaseClient)
	}

	// -------------------------------------------------------------------------
	// Runtime configuration — shared with the API process through the
	// datastore; the scheduler/publisher/analytics workers reload settings on
	// every tick so UI changes apply without restarting this process.
	// -------------------------------------------------------------------------
	runtimeCfg := runtimeconfig.New(settingsRepo, cfg, log)
	runtimeCfg.Load(context.Background())

	// -------------------------------------------------------------------------
	// Threads adapter — always constructed; workers self-gate on credentials.
	// -------------------------------------------------------------------------
	threadsAdapter := threads.NewThreadsAdapter(runtimeCfg, log)
	if runtimeCfg.ThreadsConfigured() {
		log.Info("threads adapter configured via runtime settings",
			"user_id", runtimeCfg.Str("threads_user_id", ""),
			"connected", runtimeCfg.ThreadsConnected(),
		)
	} else {
		log.Info("Threads credentials not set — publishing & analytics workers will idle until configured in Settings")
	}

	// -------------------------------------------------------------------------
	// Root context for worker goroutines
	// -------------------------------------------------------------------------
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// -------------------------------------------------------------------------
	// AMAB Scheduler — moves QUEUED → SCHEDULED
	// -------------------------------------------------------------------------
	sch := scheduler.NewScheduler(runtimeCfg, queueRepo, scheduleRepo, contentRepo, historyRepo, performanceRepo, log)
	wg.Add(1)
	go func() {
		defer wg.Done()
		sch.Run(ctx)
	}()

	// -------------------------------------------------------------------------
	// Publisher — moves SCHEDULED → PUBLISHED via the Threads API. It starts
	// unconditionally and self-gates on runtime Threads credentials.
	// -------------------------------------------------------------------------
	pub := publishing.NewPublisher(runtimeCfg, contentRepo, scheduleRepo, publishedPostRepo, historyRepo, threadsAdapter, log)
	wg.Add(1)
	go func() {
		defer wg.Done()
		pub.Run(ctx)
	}()

	// -------------------------------------------------------------------------
	// Analytics worker — collects insights → AMAB feedback loop
	// -------------------------------------------------------------------------
	aw, err := analytics.NewWorker(analytics.WorkerOptions{
		PublishedRepo:    publishedPostRepo,
		ContentRepo:      contentRepo,
		PerformanceRepo:  performanceRepo,
		HistoryRepo:      historyRepo,
		ThreadsPort:      threadsAdapter,
		UserID:           threadsAdapter.UserID(),
		Logger:           log,
		TickInterval:     10 * time.Minute,
		ProcessBatchSize: 10,
	})
	if err != nil {
		log.Error("failed to start analytics worker", "error", err)
	} else {
		wg.Add(1)
		go func() {
			defer wg.Done()
			aw.Run(ctx)
		}()
	}

	// -------------------------------------------------------------------------
	// Graceful shutdown
	// -------------------------------------------------------------------------
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Info("shutdown signal received", "signal", sig.String())

	// Signal workers to stop and give them up to 30s to finish active jobs.
	cancel()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("all workers stopped cleanly")
	case <-time.After(30 * time.Second):
		log.Warn("timed out waiting for workers to stop")
	}
}
