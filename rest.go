package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func NewRest(repo *NotificationRepo, scheduler *Scheduler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /notifications", func(w http.ResponseWriter, r *http.Request) {
		var n Notification
		if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
			slog.Error("decoding notification", "err", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := repo.Save(r.Context(), &n); err != nil {
			slog.Error("saving notification", "n", n, "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := scheduler.ScheduleNotification(r.Context(), &n); err != nil {
			slog.Error("scheduling notification", "n", n, "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	return mux
}
