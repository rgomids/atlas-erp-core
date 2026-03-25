package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Summary struct {
	Name         string  `json:"name"`
	Samples      int     `json:"samples"`
	AvgMS        float64 `json:"avg_ms"`
	P95MS        float64 `json:"p95_ms"`
	OpsPerSec    float64 `json:"ops_per_sec"`
	ErrorRatePct float64 `json:"error_rate_pct"`
}

type collector struct {
	name      string
	latencies []time.Duration
	errors    int
	total     time.Duration
}

func newCollector(name string) *collector {
	return &collector{name: name}
}

func (collector *collector) Record(duration time.Duration, err error) {
	collector.latencies = append(collector.latencies, duration)
	collector.total += duration
	if err != nil {
		collector.errors++
	}
}

func (collector *collector) Summary() Summary {
	samples := len(collector.latencies)
	if samples == 0 {
		return Summary{Name: collector.name}
	}

	ordered := append([]time.Duration(nil), collector.latencies...)
	sort.Slice(ordered, func(left int, right int) bool {
		return ordered[left] < ordered[right]
	})

	return Summary{
		Name:         collector.name,
		Samples:      samples,
		AvgMS:        durationToMilliseconds(collector.total) / float64(samples),
		P95MS:        durationToMilliseconds(percentileDuration(ordered, 0.95)),
		OpsPerSec:    float64(samples) / collector.total.Seconds(),
		ErrorRatePct: float64(collector.errors) / float64(samples) * 100,
	}
}

func percentileDuration(latencies []time.Duration, percentile float64) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	index := int(percentile*float64(len(latencies)-1) + 0.999999999)
	if index < 0 {
		index = 0
	}
	if index >= len(latencies) {
		index = len(latencies) - 1
	}

	return latencies[index]
}

func durationToMilliseconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}

type report struct {
	Phase     string    `json:"phase"`
	Generated string    `json:"generated_at"`
	Status    string    `json:"status"`
	Note      string    `json:"note,omitempty"`
	Results   []Summary `json:"results"`
}

type registry struct {
	mu      sync.Mutex
	results map[string]Summary
}

func newRegistry() *registry {
	return &registry{
		results: map[string]Summary{},
	}
}

func (registry *registry) Record(summary Summary) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	registry.results[summary.Name] = summary
}

func (registry *registry) Results() []Summary {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	results := make([]Summary, 0, len(registry.results))
	for _, summary := range registry.results {
		results = append(results, summary)
	}

	sort.Slice(results, func(left int, right int) bool {
		return results[left].Name < results[right].Name
	})

	return results
}

func writeReports(registry *registry, jsonPath string, markdownPath string, now time.Time) error {
	if registry == nil {
		registry = newRegistry()
	}

	results := registry.Results()
	report := report{
		Phase:     "Phase 7 - Portfolio Differentiation & Advanced Engineering",
		Generated: now.UTC().Format(time.RFC3339),
		Status:    "captured",
		Results:   results,
	}

	if len(results) == 0 {
		report.Status = "no_samples"
		report.Note = "No benchmark samples were collected. This usually means Docker/testcontainers-backed PostgreSQL was unavailable for the benchmark run."
	}

	if jsonPath != "" {
		if err := writeJSONReport(jsonPath, report); err != nil {
			return err
		}
	}

	if markdownPath != "" {
		if err := writeMarkdownReport(markdownPath, report); err != nil {
			return err
		}
	}

	return nil
}

func writeJSONReport(path string, report report) error {
	resolvedPath, err := resolveReportPath(path)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
		return fmt.Errorf("create benchmark report directory: %w", err)
	}

	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal benchmark json report: %w", err)
	}

	content = append(content, '\n')

	if err := os.WriteFile(resolvedPath, content, 0o644); err != nil {
		return fmt.Errorf("write benchmark json report: %w", err)
	}

	return nil
}

func writeMarkdownReport(path string, report report) error {
	resolvedPath, err := resolveReportPath(path)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
		return fmt.Errorf("create benchmark report directory: %w", err)
	}

	var builder strings.Builder
	builder.WriteString("# Phase 7 Benchmark Baseline\n\n")
	builder.WriteString(fmt.Sprintf("- Phase: `%s`\n", report.Phase))
	builder.WriteString(fmt.Sprintf("- Generated at: `%s`\n\n", report.Generated))
	builder.WriteString(fmt.Sprintf("- Status: `%s`\n", report.Status))

	if report.Note != "" {
		builder.WriteString(fmt.Sprintf("- Note: %s\n", report.Note))
	}

	builder.WriteString("\n")

	if len(report.Results) == 0 {
		builder.WriteString("No benchmark samples were captured in this run.\n\n")
		builder.WriteString("Run the benchmark command again once Docker/testcontainers-backed PostgreSQL is available to generate numeric latency and throughput evidence.\n")
	} else {
		builder.WriteString("| Benchmark | Samples | Avg (ms) | P95 (ms) | Throughput (ops/s) | Error rate (%) |\n")
		builder.WriteString("| --- | ---: | ---: | ---: | ---: | ---: |\n")

		for _, result := range report.Results {
			builder.WriteString(fmt.Sprintf(
				"| `%s` | %d | %.3f | %.3f | %.3f | %.3f |\n",
				result.Name,
				result.Samples,
				result.AvgMS,
				result.P95MS,
				result.OpsPerSec,
				result.ErrorRatePct,
			))
		}

		builder.WriteString("\n")
		builder.WriteString("These benchmarks are local portfolio evidence. They are not CI gates and should be interpreted alongside the active fault profile, machine capacity, and Docker/testcontainers availability.\n")
	}

	if err := os.WriteFile(resolvedPath, []byte(builder.String()), 0o644); err != nil {
		return fmt.Errorf("write benchmark markdown report: %w", err)
	}

	return nil
}

func resolveReportPath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}

	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve benchmark report path: get working directory: %w", err)
	}

	return resolveReportPathFrom(workingDirectory, path), nil
}

func resolveReportPathFrom(baseDirectory string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	currentDirectory := baseDirectory
	for {
		if fileInfo, err := os.Stat(filepath.Join(currentDirectory, "go.mod")); err == nil && !fileInfo.IsDir() {
			return filepath.Join(currentDirectory, path)
		}

		parentDirectory := filepath.Dir(currentDirectory)
		if parentDirectory == currentDirectory {
			return filepath.Join(baseDirectory, path)
		}

		currentDirectory = parentDirectory
	}
}
