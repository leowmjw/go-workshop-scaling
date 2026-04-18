package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/leowmjw/go-workshop-scaling/internal/workshop"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /simulate", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		replicas := queryInt(q.Get("replicas"), 2)
		cpu := queryInt(q.Get("cpu"), 85)

		next, action := workshop.RunHPA(replicas, workshop.Metrics{CPUPercent: cpu}, workshop.HPAConfig{
			TargetCPUPercent: 70,
			MinReplicas:      1,
			MaxReplicas:      10,
			StepUp:           2,
			StepDown:         1,
		})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"part":     "part1-hpa",
			"input":    map[string]any{"replicas": replicas, "cpu_pct": cpu},
			"replicas": next,
			"action":   action,
		}); err != nil {
			slog.Error("encode response", "err", err)
		}
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	addr := ":8080"
	if v := os.Getenv("PORT"); v != "" {
		addr = ":" + v
	}

	slog.Info("part1srv starting", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}

func queryInt(s string, def int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}
