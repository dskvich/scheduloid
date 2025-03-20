package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-telegram/bot"
)

type Service interface {
	Name() string
	Start(ctx context.Context) error
}

type Group []Service

func (g Group) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	startCtx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	var wg sync.WaitGroup
	errCh := make(chan error, len(g))

	wg.Add(len(g))
	for _, w := range g {
		go func(w Service) {
			defer wg.Done()
			if err := w.Start(startCtx); err != nil {
				errCh <- fmt.Errorf("%s: %w", w.Name(), err)
				cancelFn()
			}
		}(w)
	}

	<-startCtx.Done()
	wg.Wait()

	var result error
	close(errCh)
	for svcErr := range errCh {
		result = errors.Join(result, svcErr)
	}

	return result
}

type BotService struct {
	Bot *bot.Bot
}

func (b *BotService) Name() string { return "bot" }
func (b *BotService) Start(ctx context.Context) error {
	slog.Info("Starting service", "name", b.Name())
	defer slog.Info("Service stopped", "name", b.Name())

	b.Bot.Start(ctx)
	return nil
}

type RestService struct {
	Server *http.Server
}

func (r *RestService) Name() string { return "rest" }
func (r *RestService) Start(ctx context.Context) error {
	slog.Info("Starting service", "name", r.Name())
	defer slog.Info("Service stopped", "name", r.Name())

	errCh := make(chan error)
	go func() {
		err := r.Server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	select {
	case <-ctx.Done():
		return r.Server.Shutdown(ctx)
	case err := <-errCh:
		return err
	}
}

type SchedulerService struct {
	repo  *NotificationRepo
	sched *Scheduler
}

func (s *SchedulerService) Name() string { return "scheduler" }
func (s *SchedulerService) Start(ctx context.Context) error {
	slog.Info("Starting service", "name", s.Name())
	defer slog.Info("Service stopped", "name", s.Name())

	notifications, err := s.repo.GetAll(ctx)
	if err != nil {
		return err
	}

	for _, n := range notifications {
		if err := s.sched.ScheduleNotification(ctx, &n); err != nil {
			slog.Error("scheduling notification", "n", n, "err", err)
		}
	}

	s.sched.Start()
	<-ctx.Done()
	return nil
}
