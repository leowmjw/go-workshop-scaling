package main

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/leowmjw/go-workshop-scaling/internal/workshop"
)

type noopRuntime struct{}

func (noopRuntime) SetMemoryLimitMB(int64) {}
func (noopRuntime) GC()                    {}

func main() {
	recommended, _ := workshop.RecommendMemoryLimitMB(1024, workshop.Metrics{MemoryPercent: 20}, workshop.VPAConfig{
		MinAllowedMB:          256,
		MaxAllowedMB:          8192,
		UpscaleThresholdPct:   80,
		DownscaleThresholdPct: 30,
	})

	memory, action := workshop.ApplyVPA(1024, recommended, noopRuntime{})

	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"part":          "part2-vpa",
		"memoryLimitMB": memory,
		"action":        action,
	}); err != nil {
		slog.Error("encode output", "err", err)
		os.Exit(1)
	}

	slog.Info("part2 complete")
}
