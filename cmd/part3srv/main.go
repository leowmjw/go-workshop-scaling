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
		replicas := queryInt(q.Get("replicas"), 3)
		cpu := queryInt(q.Get("cpu"), 85)
		memPct := queryInt(q.Get("mem_pct"), 92)
		nodeFree := int64(queryInt(q.Get("node_free_mb"), 0))
		memLimitMB := int64(queryInt(q.Get("mem_limit_mb"), 1024))

		result := workshop.RunCombined(workshop.CombinedInput{
			Replicas:         replicas,
			MemoryLimitMB:    memLimitMB,
			NodeFreeMemoryMB: nodeFree,
			Metrics:          workshop.Metrics{CPUPercent: cpu, MemoryPercent: memPct},
			HPA: workshop.HPAConfig{
				TargetCPUPercent: 70,
				MinReplicas:      2,
				MaxReplicas:      10,
				StepUp:           5,
				StepDown:         1,
			},
			VPA: workshop.VPAConfig{
				MinAllowedMB:          256,
				MaxAllowedMB:          8192,
				UpscaleThresholdPct:   80,
				DownscaleThresholdPct: 30,
			},
		}, noopRuntime{})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"part":   "part3-hpa-vpa",
			"input":  map[string]any{"replicas": replicas, "cpu_pct": cpu, "mem_pct": memPct, "node_free_mb": nodeFree},
			"result": result,
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

	slog.Info("part3srv starting", "addr", addr)
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
