package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v9"
	"github.com/go-telegram/bot"
	"github.com/robfig/cron/v3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/extra/bundebug"
)

const timeout = time.Second * 10

type Config struct {
	TelegramBotToken          string  `env:"TELEGRAM_BOT_TOKEN,required"`
	TelegramAuthorizedUserIDs []int64 `env:"TELEGRAM_AUTHORIZED_USER_IDS" envSeparator:" "`
	PgURL                     string  `env:"DATABASE_URL"`
	PgHost                    string  `env:"DB_HOST" envDefault:"localhost:65432"`
	Port                      string  `env:"PORT"  envDefault:"8080"`
}

func main() {
	if err := runMain(); err != nil {
		slog.Error("shutting down due to error", "err", err)
		os.Exit(1)
	}
	slog.Info("shutdown complete")
}

func runMain() error {
	workerGroup, err := setupWorkers()
	if err != nil {
		return err
	}

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGHUP)
		select {
		case s := <-sigCh:
			slog.Info("shutting down due to signal", "signal", s.String())
			cancelFn()
		case <-ctx.Done():
		}
	}()

	return workerGroup.Start(ctx)
}

func setupWorkers() (Group, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parsing env config: %w", err)
	}

	sqlDB, err := NewPostgres(cfg.PgURL, cfg.PgHost)
	if err != nil {
		return nil, fmt.Errorf("creating db: %w", err)
	}

	db := bun.NewDB(sqlDB, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))
	_, err = db.NewCreateTable().Model((*Notification)(nil)).IfNotExists().Exec(context.Background())
	if err != nil {
		return nil, fmt.Errorf("creating notifications table: %w", err)
	}

	b, err := bot.New(cfg.TelegramBotToken)
	if err != nil {
		return nil, fmt.Errorf("creating bot: %w", err)
	}

	notifRepo := NewNotificationRepo(db)

	sched := &Scheduler{
		repo: notifRepo,
		cron: cron.New(),
		bot:  b,
	}

	botSvc := &BotService{b}
	schedulerSvc := &SchedulerService{
		repo:  NewNotificationRepo(db),
		sched: sched,
	}
	restSvc := &RestService{
		Server: &http.Server{
			Addr:              fmt.Sprintf(":%s", cfg.Port),
			Handler:           NewRest(notifRepo, sched),
			ReadHeaderTimeout: timeout,
		},
	}

	var svcGroup Group
	svcGroup = append(svcGroup, botSvc, schedulerSvc, restSvc)

	return svcGroup, nil
}
