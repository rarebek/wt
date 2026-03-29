#!/bin/bash
# experiment.sh — Run one experiment iteration for wt autoresearch
# Equivalent to Karpathy's `uv run train.py` but for a Go framework
set -e

echo "=== wt autoresearch experiment ==="
echo "started: $(date)"
echo ""

# Step 1: Build
echo "--- build ---"
if go build ./... 2>&1; then
    echo "build: pass"
else
    echo "build: FAIL"
    echo "---"
    echo "tests_passed:     0"
    echo "tests_failed:     1"
    echo "benchmark_echo:   0"
    echo "benchmark_allocs: 0"
    echo "go_vet:           fail"
    echo "lines_of_code:    0"
    echo "files:            0"
    exit 1
fi

# Step 2: Vet
echo ""
echo "--- vet ---"
if go vet ./... 2>&1; then
    VET="clean"
    echo "vet: clean"
else
    VET="fail"
    echo "vet: WARNINGS"
fi

# Step 3: Test
echo ""
echo "--- test ---"
TEST_OUTPUT=$(go test ./... -v -count=1 -timeout=120s 2>&1)
TESTS_PASSED=$(echo "$TEST_OUTPUT" | grep -c "^--- PASS" || true)
TESTS_FAILED=$(echo "$TEST_OUTPUT" | grep -c "^--- FAIL" || true)

if echo "$TEST_OUTPUT" | grep -q "^FAIL"; then
    echo "tests: SOME FAILED"
    echo "$TEST_OUTPUT" | grep "^FAIL" || true
else
    echo "tests: all pass"
fi
echo "passed: $TESTS_PASSED"
echo "failed: $TESTS_FAILED"

# Step 4: Benchmark (only the key echo benchmark for speed)
echo ""
echo "--- benchmark ---"
BENCH_OUTPUT=$(go test -bench=BenchmarkWT_Echo_64B -benchmem -count=1 -timeout=60s 2>&1)
BENCH_NS=$(echo "$BENCH_OUTPUT" | grep "BenchmarkWT_Echo_64B" | awk '{print $3}' | sed 's/[^0-9.]//g' || echo "0")
BENCH_ALLOCS=$(echo "$BENCH_OUTPUT" | grep "BenchmarkWT_Echo_64B" | awk '{print $7}' | sed 's/[^0-9]//g' || echo "0")

if [ -z "$BENCH_NS" ]; then BENCH_NS=0; fi
if [ -z "$BENCH_ALLOCS" ]; then BENCH_ALLOCS=0; fi

echo "echo_ns: $BENCH_NS"
echo "echo_allocs: $BENCH_ALLOCS"

# Step 5: Code stats
echo ""
echo "--- stats ---"
LOC=$(find . -name "*.go" -not -path "*/.github/*" -exec cat {} + | wc -l)
FILES=$(find . -name "*.go" -not -path "*/.github/*" | wc -l)
echo "lines: $LOC"
echo "files: $FILES"

# Summary
echo ""
echo "---"
echo "tests_passed:     $TESTS_PASSED"
echo "tests_failed:     $TESTS_FAILED"
echo "benchmark_echo:   $BENCH_NS"
echo "benchmark_allocs: $BENCH_ALLOCS"
echo "go_vet:           $VET"
echo "lines_of_code:    $LOC"
echo "files:            $FILES"
echo "finished: $(date)"
