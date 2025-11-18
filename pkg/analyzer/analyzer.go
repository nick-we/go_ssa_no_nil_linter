package analyzer

import "golang.org/x/tools/go/analysis"

// NewAnalyzer returns a placeholder analyzer so dependencies resolve while the implementation is built.
func NewAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "grpcnil",
		Doc:  "detect nil values in gRPC response messages",
		Run: func(pass *analysis.Pass) (any, error) {
			return nil, nil
		},
	}
}
