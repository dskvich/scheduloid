package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-telegram/bot"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	repo *NotificationRepo
	cron *cron.Cron
	bot  *bot.Bot
}

func (s *Scheduler) Start() {
	s.cron.Start()
}

func (s *Scheduler) ScheduleNotification(ctx context.Context, n *Notification) error {
	detachedCtx := context.WithoutCancel(ctx)

	if n.DateTimeSchedule == "" && n.CronSchedule == "" {
		slog.Info("No schedule defined, skipping notification", "n", n)
		return nil
	}

	if n.DateTimeSchedule != "" {
		loc, err := time.LoadLocation("Europe/Istanbul")
		if err != nil {
			return fmt.Errorf("loading location: %w", err)
		}

		dt, err := time.ParseInLocation("2006-01-02 15:04:05", n.DateTimeSchedule, loc)
		if err != nil {
			return fmt.Errorf("invalid date format '%s': %w", n.DateTimeSchedule, err)
		}

		now := time.Now().In(loc)
		if dt.Before(now) {
			slog.Info("skipping notification", "n", n, "now", now)
			return nil
		}

		time.AfterFunc(dt.Sub(now), func() {
			_, err := s.bot.SendMessage(detachedCtx, &bot.SendMessageParams{
				ChatID:          n.ChatID,
				MessageThreadID: n.TopicID,
				Text:            n.Message,
			})
			if err != nil {
				slog.Error("sending message", "n", n, "err", err)
			}
		})
	}

	if n.CronSchedule != "" {
		_, err := s.cron.AddFunc(n.CronSchedule, func() {
			_, err := s.bot.SendMessage(detachedCtx, &bot.SendMessageParams{
				ChatID:          n.ChatID,
				MessageThreadID: n.TopicID,
				Text:            n.Message,
			})
			if err != nil {
				slog.Error("sending message", "n", n, "err", err)
			}
		})
		if err != nil {
			return fmt.Errorf("adding cron job: %w", err)
		}
	}

	return nil
}
