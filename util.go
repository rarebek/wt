package wt

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"runtime"
	"strings"
)

// ServerInfo returns information about the framework and runtime.
func ServerInfo() map[string]string {
	return map[string]string{
		"framework":  "wt",
		"version":    Version,
		"go_version": runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"cpus":       fmt.Sprintf("%d", runtime.NumCPU()),
	}
}

// Hash returns the SHA-256 hash of the given data as a hex string.
func Hash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// JoinPath joins path segments with / separators, cleaning doubles.
func JoinPath(segments ...string) string {
	joined := strings.Join(segments, "/")
	// Clean double slashes
	for strings.Contains(joined, "//") {
		joined = strings.ReplaceAll(joined, "//", "/")
	}
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	return strings.TrimSuffix(joined, "/")
}

// Must panics if err is non-nil. Useful for initialization.
func Must[T any](val T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("wt: %v", err))
	}
	return val
}
