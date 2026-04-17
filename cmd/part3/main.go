package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/leowmjw/go-workshop-scaling/internal/workshop"
)

type noopRuntime struct{}

func (noopRuntime) SetMemoryLimitMB(int64) {}
func (noopRuntime) GC()                    {}

func main() {
	result := workshop.RunCombined(workshop.CombinedInput{
		Replicas:         3,
		MemoryLimitMB:    1024,
		NodeFreeMemoryMB: 0,
		Metrics: workshop.Metrics{
			CPUPercent:    85,
			MemoryPercent: 92,
		},
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

	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
		"part":     "part3-hpa-vpa",
		"replicas": result.Replicas,
		"vpa":      result.VPAStatus,
		"pod":      result.PodStatus,
		"hpa":      result.HPAAction,
	})

	log.Println("part3 complete")
}
