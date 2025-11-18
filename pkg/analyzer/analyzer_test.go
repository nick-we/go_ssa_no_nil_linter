package analyzer_test

import (
	"testing"

	"github.com/nick-we/go_ssa_no_nil_linter/pkg/analyzer"
	"golang.org/x/tools/go/analysis/analysistest"
)

// TestDirectNilAssignment verifies that the analyzer flags a direct assignment
// of a maybe-nil value into a non-optional proto field of a gRPC response.
func TestDirectNilAssignment(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, analyzer.NewAnalyzer(), "directnil")
}

// TestListNilAssignment verifies that the analyzer flags nil elements assigned
// into a repeated field (slice of message pointers) in a gRPC response.
func TestListNilAssignment(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, analyzer.NewAnalyzer(), "listnil")
}
