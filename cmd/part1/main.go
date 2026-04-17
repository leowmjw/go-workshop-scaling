package main

import (
	"encoding/json"
	"log"
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

	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
		"part":     "part1-hpa",
		"replicas": replicas,
		"action":   action,
	})

	log.Println("part1 complete")
}
