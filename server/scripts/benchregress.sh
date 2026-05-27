#!/usr/bin/env bash
# bench-regress smoke gate: run a small seeded benchmark and confirm the fast
# find-references path stays well ahead of the slow baseline. Guards against an
# accidental loss of the index or a reintroduced N+1. The gate fails if the
# fast p95 drifts to within 30% of the slow p95 (the optimization must keep at
# least a wide margin). Run from the server/ directory.
set -euo pipefail

OUT="${BENCH_JSON:-/tmp/bench_ci.json}"
go run ./cmd/bench -dbdir ../db -files 800 -symbols 1200 -refs 80 -iters 120 -json "$OUT" >/dev/null

slow=$(python3 -c "import json;print(json.load(open('$OUT'))['slow_p95_ms'])")
fast=$(python3 -c "import json;print(json.load(open('$OUT'))['fast_p95_ms'])")
echo "find-references p95: slow=${slow}ms fast=${fast}ms"

python3 - "$slow" "$fast" <<'PY'
import sys
slow, fast = float(sys.argv[1]), float(sys.argv[2])
limit = slow * 0.70
if fast > limit:
    print(f"regression: fast {fast}ms exceeds 70% of slow {slow}ms ({limit:.1f}ms)")
    sys.exit(1)
print(f"ok: fast {fast}ms is under the {limit:.1f}ms gate")
PY
