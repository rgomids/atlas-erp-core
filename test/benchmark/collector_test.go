package benchmark

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCollectorSummaryReportsExpectedMetrics(t *testing.T) {
	t.Parallel()

	collector := newCollector("BenchmarkCreateInvoice")
	collector.Record(10*time.Millisecond, nil)
	collector.Record(20*time.Millisecond, nil)
	collector.Record(30*time.Millisecond, errors.New("boom"))
	collector.Record(40*time.Millisecond, nil)

	summary := collector.Summary()

	if summary.Samples != 4 {
		t.Fatalf("expected 4 samples, got %d", summary.Samples)
	}

	if summary.AvgMS != 25 {
		t.Fatalf("expected avg 25ms, got %.3f", summary.AvgMS)
	}

	if summary.P95MS != 40 {
		t.Fatalf("expected p95 40ms, got %.3f", summary.P95MS)
	}

	if summary.ErrorRatePct != 25 {
		t.Fatalf("expected error rate 25%%, got %.3f", summary.ErrorRatePct)
	}

	if summary.OpsPerSec != 40 {
		t.Fatalf("expected throughput 40 ops/s, got %.3f", summary.OpsPerSec)
	}
}

func TestWriteReportsCreatesJSONAndMarkdownArtifacts(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	jsonPath := filepath.Join(tempDir, "phase7-baseline.json")
	markdownPath := filepath.Join(tempDir, "phase7-baseline.md")

	registry := newRegistry()
	registry.Record(Summary{
		Name:         "BenchmarkCreateCustomer",
		Samples:      3,
		AvgMS:        5.5,
		P95MS:        8.5,
		OpsPerSec:    180.2,
		ErrorRatePct: 0,
	})

	if err := writeReports(registry, jsonPath, markdownPath, time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("write reports: %v", err)
	}

	jsonContent, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read json report: %v", err)
	}

	if !strings.Contains(string(jsonContent), `"BenchmarkCreateCustomer"`) {
		t.Fatalf("expected json report to contain benchmark name, got %s", string(jsonContent))
	}

	if !strings.Contains(string(jsonContent), `"status": "captured"`) {
		t.Fatalf("expected json report to contain captured status, got %s", string(jsonContent))
	}

	markdownContent, err := os.ReadFile(markdownPath)
	if err != nil {
		t.Fatalf("read markdown report: %v", err)
	}

	if !strings.Contains(string(markdownContent), "| `BenchmarkCreateCustomer` |") {
		t.Fatalf("expected markdown report to contain benchmark row, got %s", string(markdownContent))
	}
}

func TestWriteReportsCreatesArtifactsWithoutSamples(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	jsonPath := filepath.Join(tempDir, "phase7-baseline.json")
	markdownPath := filepath.Join(tempDir, "phase7-baseline.md")

	if err := writeReports(newRegistry(), jsonPath, markdownPath, time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("write reports without samples: %v", err)
	}

	jsonContent, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read empty json report: %v", err)
	}

	if !strings.Contains(string(jsonContent), `"status": "no_samples"`) {
		t.Fatalf("expected empty json report to contain no_samples status, got %s", string(jsonContent))
	}

	markdownContent, err := os.ReadFile(markdownPath)
	if err != nil {
		t.Fatalf("read empty markdown report: %v", err)
	}

	if !strings.Contains(string(markdownContent), "No benchmark samples were captured in this run.") {
		t.Fatalf("expected empty markdown report to explain missing samples, got %s", string(markdownContent))
	}
}

func TestResolveReportPathFromAnchorsRelativePathsAtRepositoryRoot(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	repositoryRoot := filepath.Join(tempDir, "atlas-erp-core")
	packageDirectory := filepath.Join(repositoryRoot, "test", "benchmark")

	if err := os.MkdirAll(packageDirectory, 0o755); err != nil {
		t.Fatalf("create package directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repositoryRoot, "go.mod"), []byte("module example.com/atlas\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	resolvedPath := resolveReportPathFrom(packageDirectory, "docs/benchmarks/phase7-baseline.md")

	expectedPath := filepath.Join(repositoryRoot, "docs", "benchmarks", "phase7-baseline.md")
	if resolvedPath != expectedPath {
		t.Fatalf("expected report path %s, got %s", expectedPath, resolvedPath)
	}
}
