// Package generation contains background workers that auto-generate content.
package generation

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"kawan-threads/internal/application/runtimeconfig"
	"kawan-threads/internal/application/usecase"
	"kawan-threads/internal/domain/entity"
	"kawan-threads/internal/domain/port"
	"kawan-threads/internal/domain/repository"
	applogger "kawan-threads/internal/logger"
)

// allPillars is the pool from which Autopilot picks randomly each day.
var allPillars = []entity.ContentPillar{
	entity.PillarEdukasiHukum,
	entity.PillarTipsHukum,
	entity.PillarCeritaWarga,
	entity.PillarMythVsFact,
	entity.PillarCommunity,
	entity.PillarEngagement,
	entity.PillarTraining,
	entity.PillarMembership,
}

// goalCTAMap maps a goal key to a soft CTA hint injected into the prompt.
var goalCTAMap = map[string]string{
	"website_visit":    "Ajak audiens untuk mengunjungi website KAWAN dan pelajari lebih lanjut.",
	"training_signup":  "Dorong audiens mendaftar pelatihan hukum praktis di website KAWAN.",
	"community_growth": "Ajak audiens bergabung dan aktif bersama komunitas KAWAN.",
}

// defaultTopics provides fallback topic seeds per pillar when the TopicRepo
// returns empty (e.g. freshly seeded datastore).
var defaultTopics = map[entity.ContentPillar][]string{
	entity.PillarEdukasiHukum: {
		"Syarat sah perjanjian menurut KUHPerdata",
		"Hak konsumen berdasarkan UU Perlindungan Konsumen",
		"Cara melaporkan pelanggaran ketenagakerjaan",
		"Apa itu legal standing dan kapan kamu membutuhkannya",
	},
	entity.PillarTipsHukum: {
		"5 hal yang harus ada dalam surat perjanjian kerja",
		"Cara mengurus sertifikat tanah warisan",
		"Dokumen penting saat melapor ke kepolisian",
		"Cara meminta mediasi sengketa tanpa pengacara",
	},
	entity.PillarCeritaWarga: {
		"Kisah sukses warga memperjuangkan hak tanah mereka",
		"Pengalaman nyata menghadapi PHK tidak adil",
		"Cerita UMKM yang selamat dari jebakan kontrak",
	},
	entity.PillarMythVsFact: {
		"Mitos: Hanya orang kaya yang bisa pakai pengacara",
		"Mitos: Kontrak verbal tidak sah secara hukum",
		"Mitos: Sertifikat tanah tidak bisa digugat",
	},
	entity.PillarCommunity: {
		"Mengapa komunitas hukum penting bagi masyarakat",
		"Gerakan literasi hukum untuk generasi muda",
	},
	entity.PillarEngagement: {
		"Apa pertanyaan hukum yang paling sering kamu hadapi?",
		"Pernahkah kamu merasa dirugikan tapi tidak tahu harus lapor ke mana?",
	},
	entity.PillarTraining: {
		"Pelatihan kontrak kerja untuk freelancer",
		"Workshop hak konsumen untuk pelaku UMKM",
		"Kelas online: memahami sengketa tanah dari awal",
	},
	entity.PillarMembership: {
		"Keuntungan bergabung sebagai anggota KAWAN",
		"Akses konsultasi hukum eksklusif untuk member",
	},
}

// formats defines the mix of content formats used by autopilot.
// 70% single, 30% thread series — matches typical Threads engagement patterns.
var formats = []entity.ContentFormat{
	entity.FormatSinglePost,
	entity.FormatSinglePost,
	entity.FormatSinglePost,
	entity.FormatSinglePost,
	entity.FormatSinglePost,
	entity.FormatSinglePost,
	entity.FormatSinglePost,
	entity.FormatThreadSeries,
	entity.FormatThreadSeries,
	entity.FormatThreadSeries,
}

// AutopilotWorker runs a daily scheduler that auto-generates draft content
// without any manual setup. It respects the runtime config settings:
//
//   - autopilot_enabled    — master switch
//   - autopilot_daily_count — number of drafts per day (default 3)
//   - autopilot_goal        — goal hint for CTAs
//   - autopilot_run_hour    — local hour to trigger (default 7)
type AutopilotWorker struct {
	cfg         *runtimeconfig.Store
	contentUC   *usecase.ContentUseCase
	topicRepo   repository.TopicRepository
	logger      *slog.Logger
	tickInterval time.Duration
}

// AutopilotOptions bundles constructor arguments.
type AutopilotOptions struct {
	Cfg          *runtimeconfig.Store
	ContentRepo  repository.ContentRepository
	VersionRepo  repository.ContentVersionRepository
	HistoryRepo  repository.HistoryRepository
	TopicRepo    repository.TopicRepository
	AIProvider   port.AIProvider
	Logger       *slog.Logger
	TickInterval time.Duration // defaults to 1 minute
}

// NewAutopilotWorker constructs an AutopilotWorker.
func NewAutopilotWorker(opts AutopilotOptions) *AutopilotWorker {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	tick := opts.TickInterval
	if tick <= 0 {
		tick = time.Minute
	}
	uc := &usecase.ContentUseCase{
		ContentRepo: opts.ContentRepo,
		VersionRepo: opts.VersionRepo,
		HistoryRepo: opts.HistoryRepo,
		AIProvider:  opts.AIProvider,
		Logger:      opts.Logger,
	}
	return &AutopilotWorker{
		cfg:          opts.Cfg,
		contentUC:    uc,
		topicRepo:    opts.TopicRepo,
		logger:       opts.Logger,
		tickInterval: tick,
	}
}

// Run starts the autopilot loop. It checks once per tick whether it should
// fire today's generation batch, then waits until the next tick.
func (w *AutopilotWorker) Run(ctx context.Context) {
	applogger.LogEvent(w.logger, "autopilot.started", "tick", w.tickInterval.String())

	ticker := time.NewTicker(w.tickInterval)
	defer ticker.Stop()

	// Track the last date on which we fired to avoid double-running on the
	// same day if the process restarts.
	var lastFiredDate string

	for {
		select {
		case <-ctx.Done():
			applogger.LogEvent(w.logger, "autopilot.stopped")
			return
		case t := <-ticker.C:
			w.cfg.Reload(ctx)

			if !w.cfg.Bool(runtimeconfig.AutopilotEnabled, false) {
				continue
			}

			tz := w.cfg.Str(runtimeconfig.SchedulerTimezone, "Asia/Jakarta")
			loc, err := time.LoadLocation(tz)
			if err != nil {
				loc = time.UTC
			}

			local := t.In(loc)
			runHour := w.cfg.Int(runtimeconfig.AutopilotRunHour, 7)
			today := local.Format("2006-01-02")

			// Fire only once per day, at the configured hour.
			if local.Hour() == runHour && lastFiredDate != today {
				lastFiredDate = today
				applogger.LogEvent(w.logger, "autopilot.run_started", "date", today, "hour", runHour)
				w.generateBatch(ctx)
			}
		}
	}
}

// generateBatch generates autopilot_daily_count draft content items.
func (w *AutopilotWorker) generateBatch(ctx context.Context) {
	count := w.cfg.Int(runtimeconfig.AutopilotDailyCount, 3)
	goal := w.cfg.Str(runtimeconfig.AutopilotGoal, "website_visit")
	cta := goalCTAMap[goal]
	if cta == "" {
		cta = goalCTAMap["website_visit"]
	}

	// Shuffle pillars for variety — pick count unique pillars if possible.
	pillars := pickPillars(count)

	applogger.LogEvent(w.logger, "autopilot.batch_start",
		"count", count,
		"goal", goal,
	)

	generated := 0
	for i := 0; i < count; i++ {
		pillar := pillars[i%len(pillars)]
		topic := w.pickTopic(ctx, pillar)
		format := formats[rand.Intn(len(formats))] //nolint:gosec

		req := port.GenerateContentRequest{
			Pillar:   pillar,
			Topic:    topic,
			Audience: "Gen Z dan Millennial (18-35 tahun)",
			Tone:     "friendly, edukatif, tidak menggurui",
			Format:   format,
			CTA:      cta,
		}

		applogger.LogEvent(w.logger, "autopilot.generating",
			"pillar", string(pillar),
			"topic", topic,
			"format", string(format),
		)

		content, err := w.contentUC.GenerateContent(ctx, req)
		if err != nil {
			applogger.LogEvent(w.logger, "autopilot.generate_error",
				"pillar", string(pillar),
				"topic", topic,
				"error", err.Error(),
			)
			continue
		}

		generated++
		applogger.LogEvent(w.logger, "autopilot.draft_created",
			"content_id", content.ID,
			"pillar", string(content.Pillar),
			"format", string(content.Format),
			"quality", content.Quality.Relevance,
		)

		// Stagger requests to avoid rate-limiting Gemini API.
		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
		}
	}

	applogger.LogEvent(w.logger, "autopilot.batch_done",
		"requested", count,
		"generated", generated,
	)
}

// pickTopic returns a topic string for the given pillar. It queries the
// TopicRepository first; if empty, falls back to the built-in seed list.
func (w *AutopilotWorker) pickTopic(ctx context.Context, pillar entity.ContentPillar) string {
	if w.topicRepo != nil {
		topics, err := w.topicRepo.FindAll(ctx)
		if err == nil {
			// Filter to matching pillar and active.
			var matching []string
			for _, t := range topics {
				if t.Pillar == pillar && t.Active {
					matching = append(matching, t.Name)
				}
			}
			if len(matching) > 0 {
				return matching[rand.Intn(len(matching))] //nolint:gosec
			}
		}
	}

	// Fallback to built-in seeds.
	seeds := defaultTopics[pillar]
	if len(seeds) == 0 {
		return string(pillar) + " — topik umum"
	}
	return seeds[rand.Intn(len(seeds))] //nolint:gosec
}

// pickPillars returns count pillar values shuffled, cycling if count > len.
func pickPillars(count int) []entity.ContentPillar {
	pool := make([]entity.ContentPillar, len(allPillars))
	copy(pool, allPillars)
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] }) //nolint:gosec

	result := make([]entity.ContentPillar, 0, count)
	for len(result) < count {
		result = append(result, pool[len(result)%len(pool)])
	}
	return result
}

// newUUID generates a UUID string (wrapper for testability).
func newUUID() string { return uuid.New().String() }
