package wt

import "testing"

func TestCompressionStatsRatio(t *testing.T) {
	stats := CompressionStats{
		RawBytes:        1000,
		CompressedBytes: 300,
	}

	ratio := stats.Ratio()
	if ratio != 0.3 {
		t.Errorf("expected 0.3, got %f", ratio)
	}
}

func TestCompressionStatsZero(t *testing.T) {
	stats := CompressionStats{}
	if stats.Ratio() != 0 {
		t.Error("zero raw bytes should give 0 ratio")
	}
}
