.PHONY: build test bench fuzz vet clean

build:
	go build ./...

test:
	go test ./... -v -count=1 -timeout=60s

bench:
	go test -bench=. -benchmem -count=3 -timeout=120s

fuzz:
	go test -fuzz=FuzzMatchPattern -fuzztime=30s
	go test -fuzz=FuzzExtractParamNames -fuzztime=15s
	go test -fuzz=FuzzCountSegments -fuzztime=15s

vet:
	go vet ./...

clean:
	go clean -testcache

all: vet build test
