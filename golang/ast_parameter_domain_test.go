// Licensed under the MIT license.
package g3

import (
	"errors"
	"math"
	"testing"
)

func TestASTParameterDoubleHashSourceDomain(t *testing.T) {
	// Direct scalar-record fixture isolates conversion from eager construction;
	// it invokes the actual production parameterHash, with no substitute manager.
	m := &ASTManager{origin: 7, nextParameter: 2, parameters: []*astParameterPage{new(astParameterPage)}}
	p := ASTParameter{origin: 7, slot: 1, generation: 1}
	for _, tc := range []struct {
		v       float64
		hash    uint32
		invalid bool
	}{
		{-0.5, 4, false}, {math.Copysign(0, -1), 4, false}, {0.5, 4, false},
		{12.9, (12 << 2) | 4, false}, {4294967295.75, (math.MaxUint32<<2)&math.MaxUint32 | 4, false},
		{-1, 0, true}, {4294967296, 0, true}, {math.NaN(), 0, true}, {math.Inf(1), 0, true},
	} {
		*m.parameterAt(1) = astParameterRecord{generation: 1, live: true, kind: ASTParameterDouble, double: tc.v}
		got, e := m.parameterHash(p)
		if tc.invalid {
			if !errors.Is(e, ErrASTDomain) {
				t.Fatal(tc.v, got, e)
			}
		} else if e != nil || got != tc.hash {
			t.Fatal(tc.v, got, e, tc.hash)
		}
	}
}
