package rewriter_test

// Source: src/ast/rewriter/rewriter.h:365-388.

import (
	"github.com/glycerine/g3/pkg/ast/rewriter"
	"testing"
)

// Compile the configuration contract from another package, as model requires.
type externalTraversalConfiguration struct{ rewriter.DefaultConfiguration }

var _ rewriter.Configuration = (*externalTraversalConfiguration)(nil)

func TestTraversalExternalDefaultConfiguration(t *testing.T) {
	c := externalTraversalConfiguration{}
	exceeded, e := c.MaxStepsExceeded(123)
	if e != nil || exceeded || !c.CacheResults() || c.CacheAllResults() || !c.RewritePatterns() {
		t.Fatal("source configuration defaults", exceeded, e)
	}
}
