#!/bin/bash
# experiment.sh — single autoresearch iteration (~45s on i9-14900K)

# Build + vet
if ! go build ./... 2>/dev/null; then
    echo "---"; echo "tests_passed: 0"; echo "tests_failed: 1"; echo "benchmark_echo: 0"; exit 1
fi
VET="clean"; go vet ./... 2>/dev/null || VET="fail"

# Test (verbose, single run)
T=$(go test ./... -v -count=1 -timeout=90s 2>&1)
PASS=$(echo "$T" | grep "PASS:" | wc -l)
FAIL=$(echo "$T" | grep "^FAIL" | wc -l)

# Bench
B=$(go test -run=NOMATCH -bench=BenchmarkWT_Echo_64B -benchmem -count=1 -timeout=20s 2>&1 | grep BenchmarkWT_Echo)
NS=$(echo "$B" | awk '{print int($3)}'); : ${NS:=0}
ALLOCS=$(echo "$B" | awk '{print $5}' | tr -dc '0-9'); : ${ALLOCS:=0}

echo "---"
echo "tests_passed:     $PASS"
echo "tests_failed:     $FAIL"
echo "benchmark_echo:   $NS"
echo "benchmark_allocs: $ALLOCS"
echo "go_vet:           $VET"
echo "finished:         $(date +%H:%M:%S)"
