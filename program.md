# autoresearch — wt WebTransport Framework

This is an experiment to have the LLM autonomously improve the wt WebTransport framework.

## Setup

To set up a new experiment, work with the user to:

1. **Agree on a run tag**: propose a tag based on today's date (e.g. `mar29`). The branch `autoresearch/<tag>` must not already exist — this is a fresh run.
2. **Create the branch**: `git checkout -b autoresearch/<tag>` from current main.
3. **Read the in-scope files**: The repo has ~190 files. Read these for context:
   - `README.md` — what the framework does
   - `wt.go` — server core
   - `router.go` — routing
   - `context.go` — session context
   - `middleware.go` — middleware chain
   - `stream.go` — stream types
   - Run `go test ./... -count=1` to verify baseline
   - Run `go test -bench=BenchmarkWT_Echo_64B -benchmem -count=1` for baseline perf
4. **Initialize results.tsv**: Create `results.tsv` with just the header row.
5. **Confirm and go**: Confirm setup looks good.

Once you get confirmation, kick off the experimentation.

## Experimentation

Each experiment runs locally. The experiment script builds, tests, and benchmarks:

```bash
cd wt && ./experiment.sh
```

**What you CAN do:**
- Add new features (middleware, utilities, helpers)
- Optimize existing code (fewer allocs, faster paths)
- Add tests (more coverage, edge cases, fuzz tests)
- Fix bugs
- Improve API ergonomics
- Add examples
- Refactor for clarity

**What you CANNOT do:**
- Break existing tests (all 237+ tests must pass)
- Remove existing public API (backwards compatibility)
- Add external dependencies beyond what's in go.mod
- Change the module path

**The goals (in priority order):**
1. **All tests pass** — `go test ./...` must exit 0
2. **More tests** — increase test count
3. **Better performance** — lower ns/op on key benchmarks
4. **Fewer allocations** — lower allocs/op
5. **New features** — useful additions
6. **Simpler code** — less complexity for same functionality

**Simplicity criterion**: All else being equal, simpler is better. Removing code and getting equal test results is a great outcome. Adding 50 lines for 1ns improvement? Not worth it. Adding 10 lines that enable a whole new use case? Worth it.

**The first run**: Your very first run should always establish the baseline.

## Output format

The experiment script prints a summary like this:

```
---
tests_passed:     237
tests_failed:     0
benchmark_echo:   55000
benchmark_allocs: 26
go_vet:           clean
lines_of_code:    16000
files:            190
```

Extract key metrics:
```
grep "^tests_passed:\|^benchmark_echo:" run.log
```

## Logging results

When an experiment is done, log it to `results.tsv` (tab-separated).

The TSV has a header row and 5 columns:

```
commit	tests	bench_ns	status	description
```

1. git commit hash (short, 7 chars)
2. number of tests passing
3. benchmark echo ns/op (BenchmarkWT_Echo_64B)
4. status: `keep`, `discard`, or `crash`
5. short text description of what this experiment tried

Example:

```
commit	tests	bench_ns	status	description
a1b2c3d	237	55000	keep	baseline
b2c3d4e	240	54200	keep	add 3 pubsub tests
c3d4e5f	240	55100	discard	try radix tree router (no improvement)
d4e5f6g	0	0	crash	broke stream framing (revert)
```

## The experiment loop

The experiment runs on a dedicated branch (e.g. `autoresearch/mar29`).

LOOP FOREVER:

1. Look at the git state: the current branch/commit we're on
2. Make an improvement to the codebase (new feature, optimization, test, bugfix)
3. `git add -A && git commit -m "description"`
4. Run the experiment: `bash experiment.sh > run.log 2>&1`
5. Read out the results: `grep "^tests_passed:\|^benchmark_echo:\|^go_vet:" run.log`
6. If grep output is empty, the run crashed. Run `tail -n 50 run.log` to read the error and fix it.
7. Record the results in results.tsv (do NOT commit results.tsv)
8. If tests still pass AND (more tests OR better perf OR useful feature), keep the commit
9. If tests broke or no improvement, `git reset --hard HEAD~1`

**Timeout**: Each experiment should take ~2 minutes (build + test + bench). If it exceeds 5 minutes, kill it.

**Crashes**: If tests fail, fix the bug and re-run. If the idea is fundamentally broken, revert and move on.

**NEVER STOP**: Once the experiment loop has begun, do NOT pause to ask the human. The human might be asleep. You are autonomous. If you run out of ideas, think harder — read the research docs in `../webtransport-research/`, look at open issues, try combining features, optimize hot paths, add edge case tests. The loop runs until the human interrupts you, period.

As an example: each experiment takes ~2 minutes, so you can run ~30/hour, ~240 overnight. The user wakes up to a better framework.
