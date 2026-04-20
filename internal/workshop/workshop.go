package workshop

import (
	"fmt"
)

const (
	HPAActionNoChange               = "NoHorizontalChange"
	HPAActionScaleOut               = "ScaleOut"
	HPAActionScaleIn                = "ScaleIn"
	HPAActionFallbackScaleOut       = "FallbackScaleOutDueToInfeasibleVPA"
	VPAStatusNoRecommendation       = "NoRecommendation"
	VPAStatusRecommendationApplied  = "RecommendationApplied"
	PodStatusStable                 = "Stable"
	PodStatusResizedInPlace         = "ResizedInPlace"
	PodStatusInfeasible             = "Infeasible"
	VPAApplyActionNoChange          = "NoVerticalChange"
	VPAApplyActionUpscaledInPlace   = "UpscaledInPlace"
	VPAApplyActionDownscaledInPlace = "DownscaledInPlace"
	VPAApplyActionCappedAtMax       = "CappedAtMaxAllowed"
	VPAApplyActionRaisedToMin       = "RaisedToMinAllowed"
)

type RuntimeTuner interface {
	SetMemoryLimitMB(limitMB int64)
	GC()
}

type Metrics struct {
	CPUPercent    int
	MemoryPercent int
}

type HPAConfig struct {
	TargetCPUPercent int
	MinReplicas      int
	MaxReplicas      int
	StepUp           int
	StepDown         int
}

type VPAConfig struct {
	MinAllowedMB          int64
	MaxAllowedMB          int64
	UpscaleThresholdPct   int
	DownscaleThresholdPct int
}

type CombinedInput struct {
	Replicas         int
	MemoryLimitMB    int64
	NodeFreeMemoryMB int64
	Metrics          Metrics
	HPA              HPAConfig
	VPA              VPAConfig
}

type CombinedResult struct {
	Replicas      int
	MemoryLimitMB int64
	HPAAction     string
	VPAStatus     string
	VPAAction     string
	PodStatus     string
}

func RunHPA(replicas int, m Metrics, cfg HPAConfig) (int, string) {
	if cfg.StepUp <= 0 {
		cfg.StepUp = 1
	}
	if cfg.StepDown <= 0 {
		cfg.StepDown = 1
	}

	if m.CPUPercent > cfg.TargetCPUPercent && replicas < cfg.MaxReplicas {
		next := replicas + cfg.StepUp
		if next > cfg.MaxReplicas {
			next = cfg.MaxReplicas
		}
		return next, HPAActionScaleOut
	}

	if m.CPUPercent < cfg.TargetCPUPercent/2 && replicas > cfg.MinReplicas {
		next := replicas - cfg.StepDown
		if next < cfg.MinReplicas {
			next = cfg.MinReplicas
		}
		return next, HPAActionScaleIn
	}

	return replicas, HPAActionNoChange
}

func RecommendMemoryLimitMB(currentLimitMB int64, m Metrics, cfg VPAConfig) (int64, string) {
	recommended := currentLimitMB
	action := VPAApplyActionNoChange

	switch {
	case m.MemoryPercent > cfg.UpscaleThresholdPct:
		recommended = currentLimitMB * 125 / 100
		action = VPAApplyActionUpscaledInPlace
	case m.MemoryPercent < cfg.DownscaleThresholdPct:
		recommended = currentLimitMB / 2
		action = VPAApplyActionDownscaledInPlace
	}

	if cfg.MaxAllowedMB > 0 && recommended > cfg.MaxAllowedMB {
		recommended = cfg.MaxAllowedMB
		action = VPAApplyActionCappedAtMax
	}
	if cfg.MinAllowedMB > 0 && recommended < cfg.MinAllowedMB {
		recommended = cfg.MinAllowedMB
		action = VPAApplyActionRaisedToMin
	}

	return recommended, action
}

func ApplyVPA(currentLimitMB, proposedLimitMB int64, tuner RuntimeTuner) (int64, string) {
	if proposedLimitMB == currentLimitMB {
		return currentLimitMB, VPAApplyActionNoChange
	}

	if tuner != nil {
		tuner.SetMemoryLimitMB(proposedLimitMB)
		if proposedLimitMB < currentLimitMB {
			tuner.GC()
		}
	}

	if proposedLimitMB < currentLimitMB {
		return proposedLimitMB, VPAApplyActionDownscaledInPlace
	}
	return proposedLimitMB, VPAApplyActionUpscaledInPlace
}

func RunCombined(input CombinedInput, tuner RuntimeTuner) CombinedResult {
	replicas, hpaAction := RunHPA(input.Replicas, input.Metrics, input.HPA)
	recommendedLimit, recommendationAction := RecommendMemoryLimitMB(input.MemoryLimitMB, input.Metrics, input.VPA)

	result := CombinedResult{
		Replicas:      replicas,
		MemoryLimitMB: input.MemoryLimitMB,
		HPAAction:     hpaAction,
		VPAStatus:     VPAStatusNoRecommendation,
		VPAAction:     recommendationAction,
		PodStatus:     PodStatusStable,
	}

	if recommendedLimit == input.MemoryLimitMB {
		return result
	}

	result.VPAStatus = VPAStatusRecommendationApplied

	delta := recommendedLimit - input.MemoryLimitMB
	if delta > 0 && delta > input.NodeFreeMemoryMB {
		result.PodStatus = PodStatusInfeasible
		result.VPAAction = fmt.Sprintf("%s(InPlaceOrRecreate)", recommendationAction)
		if result.Replicas < input.HPA.MaxReplicas {
			fallbackStep := input.HPA.StepUp
			if fallbackStep <= 0 {
				fallbackStep = 1
			}
			result.Replicas += fallbackStep
			if result.Replicas > input.HPA.MaxReplicas {
				result.Replicas = input.HPA.MaxReplicas
			}
			if result.HPAAction == HPAActionNoChange {
				result.HPAAction = HPAActionFallbackScaleOut
			} else {
				result.HPAAction = result.HPAAction + "+" + HPAActionFallbackScaleOut
			}
		}
		return result
	}

	newLimit, applyAction := ApplyVPA(input.MemoryLimitMB, recommendedLimit, tuner)
	result.MemoryLimitMB = newLimit
	result.VPAAction = applyAction
	result.PodStatus = PodStatusResizedInPlace
	return result
}
