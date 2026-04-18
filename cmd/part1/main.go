package main

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/leowmjw/go-workshop-scaling/internal/workshop"
)

func main() {
	replicas, action := workshop.RunHPA(2, workshop.Metrics{CPUPercent: 86}, workshop.HPAConfig{
		TargetCPUPercent: 70,
		MinReplicas:      1,
		MaxReplicas:      10,
		StepUp:           2,
		StepDown:         1,
	})

	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"part":     "part1-hpa",
		"replicas": replicas,
		"action":   action,
	}); err != nil {
		slog.Error("encode output", "err", err)
		os.Exit(1)
	}

	slog.Info("part1 complete")
}
