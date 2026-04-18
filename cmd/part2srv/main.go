package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/leowmjw/go-workshop-scaling/internal/workshop"
)

type noopRuntime struct{}

func (noopRuntime) SetMemoryLimitMB(int64) {}
func (noopRuntime) GC()                    {}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /simulate", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		currentMB := int64(queryInt(q.Get("current_mb"), 1024))
		memPct := queryInt(q.Get("mem_pct"), 20)

		cfg := workshop.VPAConfig{
			MinAllowedMB:          256,
			MaxAllowedMB:          8192,
			UpscaleThresholdPct:   80,
			DownscaleThresholdPct: 30,
		}

		recommended, _ := workshop.RecommendMemoryLimitMB(currentMB, workshop.Metrics{MemoryPercent: memPct}, cfg)
		newLimit, action := workshop.ApplyVPA(currentMB, recommended, noopRuntime{})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"part":   "part2-vpa",
			"input":  map[string]any{"current_mb": currentMB, "mem_pct": memPct},
			"limit":  newLimit,
			"action": action,
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

	slog.Info("part2srv starting", "addr", addr)
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
