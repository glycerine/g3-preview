// Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"strings"
	"testing"
)

func gparamsDescriptionBytes(t *testing.T, r *arena.BytesRegion, s string) arena.BytesRegionRef {
	t.Helper()
	ref, e := r.Allocate(uint32(len(s) + 1))
	paramsMust(t, e)
	for i := range s {
		paramsMust(t, r.Write(ref, uint32(i), s[i]))
	}
	paramsMust(t, r.Write(ref, uint32(len(s)), 0))
	return ref
}
func TestGlobalParamsModuleDescriptionBorrowAndDisplayBranches(t *testing.T) {
	g, _, _, b := gparamsRealRegistry(t)
	paramsMust(t, g.ensureRegistered())
	rb := arena.NewBudget(^uint64(0))
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{Descriptors: arena.Config{Budget: rb, PageSize: 2}, PageSize: 64})
	paramsMust(t, e)
	defer func() {
		r.Close()
		if rb.Used() != 0 {
			t.Fatal(rb.Used())
		}
	}()
	ref := gparamsDescriptionBytes(t, r, "prefixdescription\x00ignored")
	paramsMust(t, g.RegisterModuleDescription("sat", r, ref, 6))
	out, e := NewStringBuffer64(StringBufferConfig{Budget: b})
	paramsMust(t, e)
	defer out.Close()
	paramsMust(t, g.DisplayModulesInto(out))
	got, e := out.CStr()
	paramsMust(t, e)
	if got != "[module] sat, description: description\n" {
		t.Fatal(got)
	}
	// The source retains char*, so mutation is visible without registration.
	paramsMust(t, r.Write(ref, 6, 'D'))
	paramsMust(t, out.Reset())
	paramsMust(t, g.DisplayModuleInto(out, "sat"))
	got, e = out.CStr()
	paramsMust(t, e)
	if !strings.HasPrefix(got, "[module] sat, description: Description\n    enabled (bool)") {
		t.Fatal(got)
	}
	paramsMust(t, out.Reset())
	paramsMust(t, g.DisplayInto(out, 0, true, false))
	got, e = out.CStr()
	paramsMust(t, e)
	if !strings.HasPrefix(got, "Global parameters\n    :feature-flag (bool) (default: false)\n\n[module] sat, description: Description\n") || strings.Contains(got, "To set a module") {
		t.Fatal(got)
	}
	paramsMust(t, out.Reset())
	paramsMust(t, g.DisplayInto(out, 0, false, false))
	got, e = out.CStr()
	paramsMust(t, e)
	if !strings.Contains(got, "To set a module parameter, use <module-name>.<parameter-name>=value\nExample:  pp.decimal=true\n\n") {
		t.Fatal(got)
	}
}
func TestGlobalParamsDescriptionNullAndClosedReadPrefixes(t *testing.T) {
	g, _, _, b := gparamsRealRegistry(t)
	paramsMust(t, g.ensureRegistered())
	out, e := NewStringBuffer64(StringBufferConfig{Budget: b})
	paramsMust(t, e)
	defer out.Close()
	paramsMust(t, g.RegisterModuleDescription("sat", nil, arena.BytesRegionRef{}, 0))
	if e = g.DisplayModulesInto(out); !errors.Is(e, ErrParamsStreamFailure) {
		t.Fatal(e)
	}
	got, e := out.CStr()
	paramsMust(t, e)
	if got != "[module] sat, description: " {
		t.Fatal(got)
	}
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{})
	paramsMust(t, e)
	ref := gparamsDescriptionBytes(t, r, "")
	paramsMust(t, g.RegisterModuleDescription("sat", r, ref, 0))
	paramsMust(t, out.Reset())
	paramsMust(t, g.DisplayModulesInto(out))
	got, e = out.CStr()
	paramsMust(t, e)
	if got != "[module] sat, description: \n" {
		t.Fatal(got)
	}
	r.Close()
	paramsMust(t, out.Reset())
	// Raw retained pointer registration does not read the spelling.
	paramsMust(t, g.RegisterModuleDescription("sat", r, ref, 0))
	if e = g.DisplayModulesInto(out); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	got, e = out.CStr()
	paramsMust(t, e)
	if got != "[module] sat, description: " {
		t.Fatal(got)
	}
}

func TestGlobalParamsDisplayParameterDiagnosticBeforeLookup(t *testing.T) {
	g, _, _, b := gparamsRealRegistry(t)
	out, e := NewStringBuffer64(StringBufferConfig{Budget: b})
	paramsMust(t, e)
	defer out.Close()
	debug, e := NewStringBuffer64(StringBufferConfig{Budget: b})
	paramsMust(t, e)
	defer debug.Close()
	paramsMust(t, g.DisplayParameterInto(out, debug, ":SaT.SeEd"))
	got, e := debug.CStr()
	paramsMust(t, e)
	if got != ":SaT.SeEd sat seed\n" {
		t.Fatal(got)
	}
	got, e = out.CStr()
	paramsMust(t, e)
	want := "  name:           seed\n  module:         sat\n  qualified name: sat.seed\n  type:           unsigned int\n  description:    description\n  default value:  0\n"
	if got != want {
		t.Fatalf("%q", got)
	}
	paramsMust(t, out.Reset())
	paramsMust(t, debug.Reset())
	e = g.DisplayParameterInto(out, debug, "NO.Such")
	if e == nil || e.Error() != "unknown module 'no'" {
		t.Fatal(e)
	}
	got, e = debug.CStr()
	paramsMust(t, e)
	if got != "NO.Such no such\n" {
		t.Fatal(got)
	}
	got, e = out.CStr()
	paramsMust(t, e)
	if got != "" {
		t.Fatal(got)
	}
	paramsMust(t, debug.Reset())
	paramsMust(t, g.DisplayParameterInto(out, debug, "feature_flag"))
	got, e = debug.CStr()
	paramsMust(t, e)
	if got != "feature_flag null feature_flag\n" {
		t.Fatal(got)
	}
}

func TestGlobalParamsNilStreamActualActionPrefixes(t *testing.T) {
	g, _, _, _ := gparamsRealRegistry(t)
	// Header emission precedes lazy registration.
	if e := g.DisplayInto(nil, 0, false, true); !errors.Is(e, ErrStringBufferOwner) {
		t.Fatal(e)
	}
	if g.modulesRegistered {
		t.Fatal("registration ran before failed header action")
	}
	// Module lookup/registration precedes the first row output action.
	if e := g.DisplayModulesInto(nil); !errors.Is(e, ErrStringBufferOwner) {
		t.Fatal(e)
	}
	if !g.modulesRegistered || len(g.descriptors) != 1 {
		t.Fatal("missing registration prefix")
	}
	if e := g.DisplayModuleInto(nil, "missing"); e == nil || e.Error() != "unknown module 'missing'" {
		t.Fatal(e)
	}
	if e := g.DisplayModuleInto(nil, "sat"); !errors.Is(e, ErrStringBufferOwner) {
		t.Fatal(e)
	}
	ctx, sy, _ := paramsTestOwner(t)
	b := arena.NewBudget(^uint64(0))
	cfg := gparamsTestConfig(b)
	cfg.Register = func(GlobalParamsRegistrar) error { return nil }
	empty, e := NewGlobalParams(sy, ctx, cfg)
	paramsMust(t, e)
	defer func() {
		paramsMust(t, empty.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	// With no source insertion into an empty output stream, nil is not used.
	paramsMust(t, empty.DisplayModulesInto(nil))
}
