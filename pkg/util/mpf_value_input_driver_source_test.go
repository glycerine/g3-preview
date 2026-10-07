// Source: src/test/mpf.cpp:23-82; src/util/mpf.cpp:69-181.
// Licensed under the MIT license. Source authors: Leonardo de Moura; Christoph Wintersteiger.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// This uses the literal int/double/float SET calls from both registered driver
// helpers. The rounding-dependent INT_MAX float assertion stays source-disabled.
func TestArbitraryFloatOriginalSetIntAndDoubleDriverAssertions(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	config := IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}}
	m, e := NewArbitraryFloatManager(config)
	serviceMust(t, e)
	var a ArbitraryFloat
	assertions := 0
	d := func(want float64) {
		t.Helper()
		got, e := m.ToDouble(&a)
		serviceMust(t, e)
		if got != want {
			t.Fatal("source ENSURE to_double", got, want)
		}
		assertions++
	}
	f := func(want float32) {
		t.Helper()
		got, e := m.ToFloat(&a)
		serviceMust(t, e)
		if got != want {
			t.Fatal("source ENSURE to_float", got, want)
		}
		assertions++
	}
	serviceMust(t, m.SetInt32(&a, 11, 53, 3))
	d(3)
	serviceMust(t, m.SetInt32(&a, 11, 53, 0))
	d(0)
	serviceMust(t, m.SetInt32(&a, 11, 53, -1))
	d(-1)
	serviceMust(t, m.SetInt32(&a, 11, 53, 2147483647))
	d(2147483647)
	serviceMust(t, m.SetInt32(&a, 11, 53, -2147483648))
	d(-2147483648)
	serviceMust(t, m.SetInt32(&a, 8, 24, 3))
	f(3)
	d(3)
	serviceMust(t, m.SetInt32(&a, 8, 24, 0))
	f(0)
	d(0)
	serviceMust(t, m.SetInt32(&a, 8, 24, -1))
	f(-1)
	d(-1)
	serviceMust(t, m.SetInt32(&a, 8, 24, -2147483648))
	f(-2147483648)
	// bug_set_int and bug_set_double each construct their own manager and
	// scoped_mpf. End the first actual value and manager before the second.
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.SourceFinalize())
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal("bug_set_int retained ownership", b.Used())
	}
	m, e = NewArbitraryFloatManager(config)
	serviceMust(t, e)
	a = ArbitraryFloat{}
	serviceMust(t, m.SetDouble(&a, 11, 53, 2.5))
	d(2.5)
	serviceMust(t, m.SetDouble(&a, 11, 53, -42.25))
	d(-42.25)
	serviceMust(t, m.SetDouble(&a, 8, 24, 2.5))
	d(2.5)
	serviceMust(t, m.SetDouble(&a, 8, 24, -42.25))
	d(-42.25)
	serviceMust(t, m.SetFloat(&a, 8, 24, 2.5))
	f(2.5)
	serviceMust(t, m.SetFloat(&a, 8, 24, -42.25))
	f(-42.25)
	if assertions != 18 {
		t.Fatal("lost original ENSURE", assertions)
	}
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.SourceFinalize())
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
