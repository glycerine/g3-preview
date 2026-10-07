// Source: src/math/polynomial/polynomial.cpp:6179-6232,6307-6344.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// Independent named physical census; borrowed views are not owning roots.
func polynomialVariablePhysical(t *testing.T, m *MultivariatePolynomialCore, maxOwners []*PolynomialMaxDegrees, posOwners []*PolynomialVariablePositions, maxScopes []*ScopedPolynomialMaxDegrees, posScopes []*ScopedPolynomialVariablePositions, external []util.Block, polys []Polynomial) {
	t.Helper()
	var fields []*PolynomialVariableDegrees
	var headers uint64
	for _, d := range maxOwners {
		if d != nil {
			headers += d.header
			fields = append(fields, &d.maxDegree, &d.variables)
		}
	}
	for _, d := range posOwners {
		if d != nil {
			headers += d.header
			fields = append(fields, &d.positions)
		}
	}
	for _, s := range maxScopes {
		if s != nil {
			headers += s.header
		}
	}
	for _, s := range posScopes {
		if s != nil {
			headers += s.header
		}
	}
	if m.queryPositions.raw != (util.Block{}) {
		external = append(external, m.queryPositions.raw)
	}
	if m.queryPositions.pending != (util.Block{}) {
		external = append(external, m.queryPositions.pending)
	}
	polynomialVariablePhysicalCore(t, m, nil, nil, nil, nil, polys, nil, external, fields, headers)
	polynomialOperationMonomialAudit(t, m, polys, nil)
}
func polynomialVariablePhysicalCore(t *testing.T, m *MultivariatePolynomialCore, owners []*PolynomialNewtonInterpolator, references []*PolynomialRef, refvectors []*PolynomialRefVector, lazy []*PolynomialNewtonVector, polys []Polynomial, handlers []*PolynomialDeletionHandler, external []util.Block, degrees []*PolynomialVariableDegrees, extraColdHeaders uint64, public ...*util.Int) {
	t.Helper()
	// The current shared census retains exact cell validation, paired failure
	// states and all actual pseudo/Power/ExactDiv scope roots. This wrapper adds
	// only the supplied helpers' actual cold headers and published/pending blocks.
	extra := extraColdHeaders
	for _, d := range degrees {
		extra += d.header
		if !d.rawDestroyed && d.raw != (util.Block{}) {
			external = append(external, d.raw)
		}
		if d.pending != (util.Block{}) {
			external = append(external, d.pending)
		}
	}
	polynomialNewtonPhysicalWithExtra(t, m, owners, references, refvectors, lazy, polys, handlers, external, extra, public...)
}
