# go-workshop-scaling

Beginner-friendly standalone Go + KWOK workshop in 3 parts:

1. **Part 1 – HPA** (`cmd/part1`): scale replicas based on CPU.
2. **Part 2 – VPA** (`cmd/part2`): right-size memory and adjust Go runtime before memory downscale.
3. **Part 3 – HPA + VPA** (`cmd/part3`): coordinate both autoscalers and handle infeasible in-place resize by falling back to horizontal scaling.

## Why this workshop

It demonstrates the modern coordination model:

- HPA handles horizontal response to traffic spikes.
- VPA handles vertical right-sizing.
- Combined flow uses `InPlaceOrRecreate`-style fallback behavior when node memory is constrained.
- Runtime adaptation is explicit: downscale calls runtime memory-limit update and GC before cgroup shrink behavior is simulated.

## Run locally

```bash
go test ./...
go run ./cmd/part1
go run ./cmd/part2
go run ./cmd/part3
```

## Run with mise + overmind + air

```bash
mise run part1:dev
mise run part2:dev
mise run part3:dev
```

## Test scenario included

`Test_HPA_VPA_Conflict_Simulation` verifies:

1. Node has `0MB` free capacity.
2. VPA recommendation increases memory.
3. VPA status is `RecommendationApplied` and Pod status is `Infeasible`.
4. HPA scales out to preserve availability.
