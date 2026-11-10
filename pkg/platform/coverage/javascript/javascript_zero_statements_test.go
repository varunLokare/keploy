package javascript

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func writeZeroStatementsCoverageFixture(t *testing.T, coverageJSON string) string {
	t.Helper()
	dir := t.TempDir()

	coveragePath := filepath.Join(dir, "coverage.json")
	if err := os.WriteFile(coveragePath, []byte(coverageJSON), 0o644); err != nil {
		t.Fatalf("write coverage file: %v", err)
	}

	processInfoDir := filepath.Join(dir, ".nyc_output", "processinfo")
	if err := os.MkdirAll(processInfoDir, 0o755); err != nil {
		t.Fatalf("create processinfo dir: %v", err)
	}
	processInfo := fmt.Sprintf(`{"coverageFilename":%q,"files":["index.js"]}`, coveragePath)
	if err := os.WriteFile(filepath.Join(processInfoDir, "1.json"), []byte(processInfo), 0o644); err != nil {
		t.Fatalf("write processinfo: %v", err)
	}
	return dir
}

func TestFormatCoveragePercent(t *testing.T) {
	tests := []struct {
		name           string
		covered, total int
		want           string
	}{
		{name: "no statements", covered: 0, total: 0, want: "0.00%"},
		{name: "uncovered statements", covered: 0, total: 3, want: "0.00%"},
		{name: "all covered", covered: 3, total: 3, want: "100.00%"},
		{name: "partial", covered: 1, total: 3, want: "33.33%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatCoveragePercent(tt.covered, tt.total); got != tt.want {
				t.Fatalf("formatCoveragePercent(%d, %d) = %q, want %q", tt.covered, tt.total, got, tt.want)
			}
		})
	}
}

func TestGetCoverage_NoStatementsReportsZeroPercent(t *testing.T) {
	tests := []struct {
		name         string
		coverage     string
		wantTotalCov string
		wantFileCov  map[string]string
		wantTotal    int
		wantCovered  int
	}{
		{
			name:         "file with no statements",
			coverage:     `{"/app/config.ts":{"path":"/app/config.ts","statementMap":{},"s":{}}}`,
			wantTotalCov: "0.00%",
			wantFileCov:  map[string]string{"/app/config.ts": "0.00%"},
		},
		{
			name:         "coverage with no files",
			coverage:     `{}`,
			wantTotalCov: "0.00%",
			wantFileCov:  map[string]string{},
		},
		{
			name: "empty file alongside a partially covered file",
			coverage: `{
				"/app/config.ts":{"statementMap":{},"s":{}},
				"/app/index.js":{"statementMap":{"0":{},"1":{}},"s":{"0":1,"1":0}}
			}`,
			wantTotalCov: "50.00%",
			wantFileCov:  map[string]string{"/app/config.ts": "0.00%", "/app/index.js": "50.00%"},
			wantTotal:    2,
			wantCovered:  1,
		},
		{
			name:         "statements that never ran",
			coverage:     `{"/app/index.js":{"statementMap":{"0":{},"1":{}},"s":{"0":0,"1":0}}}`,
			wantTotalCov: "0.00%",
			wantFileCov:  map[string]string{"/app/index.js": "0.00%"},
			wantTotal:    2,
			wantCovered:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(writeZeroStatementsCoverageFixture(t, tt.coverage))

			j := New(t.Context(), zap.NewNop(), nil, "node index.js")
			got, err := j.GetCoverage()
			if err != nil {
				t.Fatalf("GetCoverage(): %v", err)
			}
			if got.TotalCov != tt.wantTotalCov {
				t.Fatalf("TotalCov = %q, want %q", got.TotalCov, tt.wantTotalCov)
			}
			if got.Loc.Total != tt.wantTotal || got.Loc.Covered != tt.wantCovered {
				t.Fatalf("Loc = %+v, want {Total:%d Covered:%d}", got.Loc, tt.wantTotal, tt.wantCovered)
			}
			if !maps.Equal(got.FileCov, tt.wantFileCov) {
				t.Fatalf("FileCov = %v, want %v", got.FileCov, tt.wantFileCov)
			}
		})
	}
}
