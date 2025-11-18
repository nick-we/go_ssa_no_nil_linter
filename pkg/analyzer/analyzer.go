package analyzer

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
)

// NewAnalyzer constructs the top-level analysis.Analyzer used by the CLI.
func NewAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "grpcnil",
		Doc:  "detect nil values in gRPC response messages",
		Run:  run,
		Requires: []*analysis.Analyzer{
			buildssa.Analyzer,
		},
	}
}

// run is the entry point invoked by the analysis framework for each package.
func run(pass *analysis.Pass) (any, error) {
	// Obtain SSA built by the shared buildssa pass, which includes imports like context.
	res, ok := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	if !ok || res == nil {
		return nil, nil
	}

	// Initialize core analyzers.
	protoAnalyzer := NewProtoFieldAnalyzer()
	nilAnalyzer := NewNilFlowAnalyzer()

	// Walk all source functions in this package and treat those that look like
	// gRPC handlers as analysis roots.
	for _, fn := range res.SrcFuncs {
		if h := DetectHandlerFromFunc(fn); h != nil {
			analyzeHandler(pass, protoAnalyzer, nilAnalyzer, *h)
		}
	}

	return nil, nil
}
