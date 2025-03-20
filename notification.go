package main

import (
	"context"

	"github.com/uptrace/bun"
)

type Notification struct {
	ID               int64  `json:"id" bun:",pk,autoincrement"`
	ChatID           int64  `json:"chat_id"`
	TopicID          int    `json:"topic_id"`
	Message          string `json:"message"`
	DateTimeSchedule string `json:"date_time_schedule"` // "2025-02-19 14:00:00"
	CronSchedule     string `json:"cron_schedule"`      // "0 0 14 * * 1"
}

type NotificationRepo struct {
	db *bun.DB
}

func NewNotificationRepo(db *bun.DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

func (r *NotificationRepo) Save(ctx context.Context, n *Notification) error {
	_, err := r.db.NewInsert().Model(n).Exec(ctx)
	return err
}

func (r *NotificationRepo) GetAll(ctx context.Context) ([]Notification, error) {
	var res []Notification
	err := r.db.NewSelect().Model(&res).Scan(ctx)
	return res, err
}

func (r *NotificationRepo) DeleteByID(ctx context.Context, id int64) error {
	_, err := r.db.NewDelete().Model((*Notification)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}
