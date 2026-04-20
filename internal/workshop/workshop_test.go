package workshop

import (
	"strconv"
	"testing"
)

type fakeRuntimeTuner struct {
	events []string
}

func (f *fakeRuntimeTuner) SetMemoryLimitMB(limitMB int64) {
	f.events = append(f.events, "set:"+strconv.FormatInt(limitMB, 10))
}

func (f *fakeRuntimeTuner) GC() {
	f.events = append(f.events, "gc")
}

func TestRunHPA(t *testing.T) {
	cfg := HPAConfig{TargetCPUPercent: 70, MinReplicas: 1, MaxReplicas: 10, StepUp: 2, StepDown: 1}

	next, action := RunHPA(2, Metrics{CPUPercent: 85}, cfg)
	if next != 4 || action != HPAActionScaleOut {
		t.Fatalf("expected scale out to 4, got replicas=%d action=%s", next, action)
	}

	next, action = RunHPA(3, Metrics{CPUPercent: 20}, cfg)
	if next != 2 || action != HPAActionScaleIn {
		t.Fatalf("expected scale in to 2, got replicas=%d action=%s", next, action)
	}

	next, action = RunHPA(3, Metrics{CPUPercent: 60}, cfg)
	if next != 3 || action != HPAActionNoChange {
		t.Fatalf("expected stable replicas=3, got replicas=%d action=%s", next, action)
	}
}

func TestApplyVPA_Downscale_UpdatesRuntimeBeforeGC(t *testing.T) {
	tuner := &fakeRuntimeTuner{}

	next, action := ApplyVPA(1024, 512, tuner)
	if next != 512 {
		t.Fatalf("expected memory limit 512, got %d", next)
	}
	if action != VPAApplyActionDownscaledInPlace {
		t.Fatalf("expected downscale action, got %s", action)
	}

	if len(tuner.events) != 2 {
		t.Fatalf("expected 2 runtime events, got %d", len(tuner.events))
	}
	if tuner.events[0] != "set:512" || tuner.events[1] != "gc" {
		t.Fatalf("expected runtime to set limit then run gc, got %v", tuner.events)
	}
}

func TestRecommendMemoryLimitMB_CapsAtMaxAllowed(t *testing.T) {
	recommended, action := RecommendMemoryLimitMB(4096, Metrics{MemoryPercent: 95}, VPAConfig{
		MinAllowedMB:          256,
		MaxAllowedMB:          8192,
		UpscaleThresholdPct:   80,
		DownscaleThresholdPct: 30,
	})

	if recommended != 5120 || action != VPAApplyActionUpscaledInPlace {
		t.Fatalf("unexpected recommendation: limit=%d action=%s", recommended, action)
	}

	recommended, action = RecommendMemoryLimitMB(8192, Metrics{MemoryPercent: 95}, VPAConfig{
		MinAllowedMB:          256,
		MaxAllowedMB:          8192,
		UpscaleThresholdPct:   80,
		DownscaleThresholdPct: 30,
	})

	if recommended != 8192 || action != VPAApplyActionCappedAtMax {
		t.Fatalf("expected cap at max, got limit=%d action=%s", recommended, action)
	}
}

func Test_HPA_VPA_Conflict_Simulation(t *testing.T) {
	input := CombinedInput{
		Replicas:         3,
		MemoryLimitMB:    1024,
		NodeFreeMemoryMB: 0,
		Metrics: Metrics{
			CPUPercent:    85,
			MemoryPercent: 92,
		},
		HPA: HPAConfig{
			TargetCPUPercent: 70,
			MinReplicas:      2,
			MaxReplicas:      10,
			StepUp:           5,
			StepDown:         1,
		},
		VPA: VPAConfig{
			MinAllowedMB:          256,
			MaxAllowedMB:          8192,
			UpscaleThresholdPct:   80,
			DownscaleThresholdPct: 30,
		},
	}

	result := RunCombined(input, &fakeRuntimeTuner{})

	if result.VPAStatus != VPAStatusRecommendationApplied {
		t.Fatalf("expected VPA status RecommendationApplied, got %s", result.VPAStatus)
	}
	if result.PodStatus != PodStatusInfeasible {
		t.Fatalf("expected pod status Infeasible, got %s", result.PodStatus)
	}
	if result.Replicas != 10 {
		t.Fatalf("expected HPA to scale out to 10 replicas, got %d", result.Replicas)
	}
	if result.MemoryLimitMB != 1024 {
		t.Fatalf("expected memory limit unchanged due to infeasible resize, got %d", result.MemoryLimitMB)
	}
}

func TestRunCombined_Downscale_AppliesRuntimeTuning(t *testing.T) {
	tuner := &fakeRuntimeTuner{}
	result := RunCombined(CombinedInput{
		Replicas:         2,
		MemoryLimitMB:    1024,
		NodeFreeMemoryMB: 1024,
		Metrics: Metrics{
			CPUPercent:    30,
			MemoryPercent: 20,
		},
		HPA: HPAConfig{
			TargetCPUPercent: 70,
			MinReplicas:      2,
			MaxReplicas:      10,
			StepUp:           2,
			StepDown:         1,
		},
		VPA: VPAConfig{
			MinAllowedMB:          256,
			MaxAllowedMB:          8192,
			UpscaleThresholdPct:   80,
			DownscaleThresholdPct: 30,
		},
	}, tuner)

	if result.MemoryLimitMB != 512 {
		t.Fatalf("expected downscale to 512MB, got %d", result.MemoryLimitMB)
	}
	if result.PodStatus != PodStatusResizedInPlace {
		t.Fatalf("expected in-place resize pod status, got %s", result.PodStatus)
	}
	if len(tuner.events) != 2 || tuner.events[0] != "set:512" || tuner.events[1] != "gc" {
		t.Fatalf("unexpected runtime events: %v", tuner.events)
	}
}

func TestRunCombined_NoRecommendation_StaysStable(t *testing.T) {
	result := RunCombined(CombinedInput{
		Replicas:         3,
		MemoryLimitMB:    1024,
		NodeFreeMemoryMB: 1024,
		Metrics: Metrics{
			CPUPercent:    60,
			MemoryPercent: 60,
		},
		HPA: HPAConfig{
			TargetCPUPercent: 70,
			MinReplicas:      2,
			MaxReplicas:      10,
			StepUp:           2,
			StepDown:         1,
		},
		VPA: VPAConfig{
			MinAllowedMB:          256,
			MaxAllowedMB:          8192,
			UpscaleThresholdPct:   80,
			DownscaleThresholdPct: 30,
		},
	}, nil)

	if result.VPAStatus != VPAStatusNoRecommendation {
		t.Fatalf("expected no recommendation, got %s", result.VPAStatus)
	}
	if result.PodStatus != PodStatusStable {
		t.Fatalf("expected stable pod status, got %s", result.PodStatus)
	}
	if result.HPAAction != HPAActionNoChange {
		t.Fatalf("expected no hpa action, got %s", result.HPAAction)
	}
}
