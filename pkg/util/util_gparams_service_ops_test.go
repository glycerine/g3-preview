// Source: src/util/gparams.cpp:557-631; src/util/gparams.h:36-143;
// src/util/symbol.h:96; src/util/debug.h:60-65.
// Author: Leonardo de Moura (gparams: 2012-11-29; symbol: 2006-09-11).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
package util

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func gparamsNamespaceFixture(t *testing.T) (*GlobalParamsService, *arena.Budget) {
	t.Helper()
	ctx, sy, _ := paramsTestOwner(t)
	b, cb := arena.NewBudget(1<<27), arena.NewBudget(1<<26)
	catalog, e := NewGlobalParamsCatalog(sy, catalogTestConfig(cb))
	paramsMust(t, e)
	cfg := gparamsTestConfig(b)
	cfg.Catalog = catalog
	s, e := NewGlobalParamsService(sy, ctx, cfg)
	paramsMust(t, e)
	t.Cleanup(func() {
		b.FailAfter(-1)
		paramsMust(t, s.Close())
		paramsMust(t, catalog.Close())
		if b.Used() != 0 || cb.Used() != 0 {
			t.Error("namespace declared cold roots", b.Used(), cb.Used())
		}
	})
	return s, b
}
func namespaceNullError(e error) bool {
	if GlobalParamsSourceDebug {
		return errors.Is(e, ErrGlobalParamsAssertion)
	}
	return errors.Is(e, ErrGlobalParamsUninitialized)
}
func TestGlobalParamsNamespaceNullAndSourceArgumentOrder(t *testing.T) {
	s, b := gparamsNamespaceFixture(t)
	before := s.Symbols.Stats().Symbols
	if _, e := s.SourceGetModuleName("namespace-before-null-assert"); !namespaceNullError(e) {
		t.Fatal("source get_module null", e)
	}
	if s.Symbols.Stats().Symbols != before+1 {
		t.Fatal("source symbol construction before current assertion")
	}
	before = s.Symbols.Stats().Symbols
	if e := s.SourceDisplayModuleInto(nil, "namespace-after-null-assert"); !namespaceNullError(e) {
		t.Fatal("source display_module null", e)
	}
	if s.Symbols.Stats().Symbols != before {
		t.Fatal("display_module touched symbol before current assertion")
	}
	if e := s.SourceSetSymbol(NumericalSymbol(1), "unused"); !namespaceNullError(e) {
		t.Fatal("bare_str numeric assertion preceded g_imp assertion", e)
	}
	if e := s.SourceReset(); !namespaceNullError(e) {
		t.Fatal("reset current guard", e)
	}
	gparamsServiceAudit(t, s, b)
}
func TestGlobalParamsNamespaceActualCurrentReinitAndSharedRef(t *testing.T) {
	s, b := gparamsNamespaceFixture(t)
	paramsMust(t, s.InitializeSource())
	first, e := s.Manager()
	paramsMust(t, e)
	d, e := NewParamDescrs(s.Symbols, ParamDescrsConfig{Budget: b})
	paramsMust(t, e)
	paramsMust(t, d.InsertName("namespace_extra", ParamUInt, pdLiteral("actual source extra"), pdLiteral("23"), nil))
	paramsMust(t, s.SourceRegisterGlobal(d))
	paramsMust(t, d.Close())
	paramsMust(t, s.SourceSetName("namespace_extra", "41"))
	got, e := s.SourceGetValueName("namespace_extra")
	paramsMust(t, e)
	if got != "41" {
		t.Fatal(got)
	}
	name, e := s.Symbols.Intern("namespace_extra")
	paramsMust(t, e)
	paramsMust(t, s.SourceSetSymbol(name, "42"))
	got, e = s.SourceGetValueSymbol(name)
	paramsMust(t, e)
	if got != "42" {
		t.Fatal(got)
	}
	ref, e := s.SourceGetRef()
	paramsMust(t, e)
	if ref != &first.globalParams {
		t.Fatal("get_ref cloned instead of borrowed current")
	}
	paramsMust(t, s.InitializeSource())
	second, e := s.Manager()
	paramsMust(t, e)
	if second == first {
		t.Fatal("source unconditional init")
	}
	ref, e = s.SourceGetRef()
	paramsMust(t, e)
	if ref != &second.globalParams {
		t.Fatal("namespace cached old imp")
	}
	got, e = first.GetValueName("namespace_extra")
	paramsMust(t, e)
	if got != "42" {
		t.Fatal("source overwrite destroyed predecessor")
	}
	gparamsServiceAudit(t, s, b)
}
func TestGlobalParamsNamespaceAssertionBindingAtReachedSites(t *testing.T) {
	s, b := gparamsNamespaceFixture(t)
	debug, region, db := debugFixture(t)
	_ = region
	_ = db
	paramsMust(t, s.BindNamespaceAssertions(debug))
	paramsMust(t, debug.EnableAssertions(false))
	if e := s.SourceReset(); !errors.Is(e, ErrGlobalParamsUninitialized) {
		t.Fatal("disabled assertion source raw null guard", e)
	}
	paramsMust(t, debug.Close())
	e := s.SourceReset()
	if GlobalParamsSourceDebug {
		if !errors.Is(e, arena.ErrClosed) {
			t.Fatal("actual reached closed debug binding", e)
		}
	} else if !errors.Is(e, ErrGlobalParamsUninitialized) {
		t.Fatal("release read erased assertion binding", e)
	}
	gparamsServiceAudit(t, s, b)
}

func TestGlobalParamsNamespaceEveryNullForwardLeavesUnusedArgumentsUnread(t *testing.T) {
	s, b := gparamsNamespaceFixture(t)
	actions := []struct {
		name string
		call func() error
	}{
		{"set-name", func() error { return s.SourceSetName("unused", "unused") }},
		{"set-symbol", func() error { return s.SourceSetSymbol(NumericalSymbol(1), "unused") }},
		{"get-name", func() error { _, e := s.SourceGetValueName("unused"); return e }},
		{"get-symbol", func() error { _, e := s.SourceGetValueSymbol(NumericalSymbol(1)); return e }},
		{"register-global", func() error { return s.SourceRegisterGlobal(nil) }},
		{"register-module", func() error { return s.SourceRegisterModule("unused", nil) }},
		{"register-description", func() error { return s.SourceRegisterModuleDescription("unused", nil, arena.BytesRegionRef{}, 0) }},
		{"get-module-symbol", func() error { _, e := s.SourceGetModuleSymbol(NumericalSymbol(1)); return e }},
		{"get-ref", func() error { _, e := s.SourceGetRef(); return e }},
		{"display", func() error { return s.SourceDisplayInto(nil, 0, false, true) }},
		{"display-modules", func() error { return s.SourceDisplayModulesInto(nil) }},
		{"display-module", func() error { return s.SourceDisplayModuleInto(nil, "unused") }},
		{"display-parameter", func() error { return s.SourceDisplayParameterInto(nil, nil, "unused") }},
	}
	before := s.Symbols.Stats().Symbols
	for _, a := range actions {
		t.Run(a.name, func(t *testing.T) {
			if e := a.call(); !namespaceNullError(e) {
				t.Fatal("source first assertion/pointer access", e)
			}
			gparamsServiceAudit(t, s, b)
			if s.Symbols.Stats().Symbols != before {
				t.Fatal("unused argument interned")
			}
		})
	}
}
func TestGlobalParamsNamespaceNumericBareStrAndDisabledAssertion(t *testing.T) {
	s, b := gparamsNamespaceFixture(t)
	paramsMust(t, s.InitializeSource())
	g, e := s.Manager()
	paramsMust(t, e)
	numeric := NumericalSymbol(1)
	if GlobalParamsSourceDebug {
		if _, e := s.SourceGetValueSymbol(numeric); !errors.Is(e, ErrGlobalParamsAssertion) {
			t.Fatal("bare_str actual numeric SASSERT", e)
		}
	}
	d, _, _ := debugFixture(t)
	paramsMust(t, d.EnableAssertions(false))
	paramsMust(t, s.BindNamespaceAssertions(d))
	_, want := g.GetValueName("")
	_, got := s.SourceGetValueSymbol(numeric)
	if want == nil || got == nil || want.Error() != got.Error() {
		t.Fatal("numeric bare_str did not use empty-name engine", want, got)
	}
	want = g.SetName("", "true")
	got = s.SourceSetSymbol(numeric, "true")
	if want == nil || got == nil || want.Error() != got.Error() {
		t.Fatal("numeric setter bare_str did not use empty-name engine", want, got)
	}
	gparamsServiceAudit(t, s, b)
}

func namespaceModule(t *testing.T, s *GlobalParamsService, b *arena.Budget) string {
	t.Helper()
	d, e := NewParamDescrs(s.Symbols, ParamDescrsConfig{Budget: b})
	paramsMust(t, e)
	text := strings.Repeat("actual-description-", 9)
	paramsMust(t, d.InsertName("enabled", ParamBool, pdLiteral(text), pdLiteral("false"), nil))
	paramsMust(t, s.SourceRegisterModule("namespace", d)) // real owning transfer
	return text
}
func TestGlobalParamsNamespaceModuleSnapshotDescriptionAndDisplays(t *testing.T) {
	s, b := gparamsNamespaceFixture(t)
	paramsMust(t, s.InitializeSource())
	text := namespaceModule(t, s, b)
	rb := arena.NewBudget(1 << 24)
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{Budget: rb, PageSize: 2}})
	paramsMust(t, e)
	defer func() {
		paramsMust(t, r.Close())
		if rb.Used() != 0 {
			t.Error("caller description root")
		}
	}()
	raw := gparamsDescriptionBytes(t, r, "borrowed")
	paramsMust(t, s.SourceRegisterModuleDescription("namespace", r, raw, 0))
	paramsMust(t, s.SourceSetName("namespace.enabled", "true"))
	key, e := s.Symbols.Intern("namespace")
	paramsMust(t, e)
	snapshot, e := s.SourceGetModuleSymbol(key)
	paramsMust(t, e)
	enabledName := "enabled"
	enabled, e := s.Context.GetBoolName(&snapshot, &enabledName, false)
	paramsMust(t, e)
	if !enabled {
		t.Fatal("source snapshot lost typed bool")
	}
	out := pdStreamBuffer(t, nil)
	debugOut := pdStreamBuffer(t, nil)
	paramsMust(t, s.SourceDisplayModuleInto(out, "namespace"))
	want := "[module] namespace, description: borrowed\n    enabled (bool) " + text + " (default: false)\n"
	if got := pdStreamText(t, out); got != want {
		t.Fatal("module display literal output", got, want)
	}
	paramsMust(t, r.Write(raw, 0, 'B'))
	paramsMust(t, out.Reset())
	paramsMust(t, s.SourceDisplayModuleInto(out, "namespace"))
	if got := pdStreamText(t, out); got != strings.Replace(want, "borrowed", "Borrowed", 1) {
		t.Fatal("description was copied at namespace binding", got)
	}
	paramsMust(t, out.Reset())
	paramsMust(t, s.SourceDisplayModulesInto(out))
	if !strings.Contains(pdStreamText(t, out), "[module] namespace, description: Borrowed\n") {
		t.Fatal("namespace display_modules route")
	}
	paramsMust(t, out.Reset())
	paramsMust(t, s.SourceDisplayInto(out, 0, true, false))
	if got := pdStreamText(t, out); !strings.HasPrefix(got, "Global parameters\n") || !strings.Contains(got, "    :enabled (bool) (default: false)\n") {
		t.Fatal("namespace full display", got)
	}
	paramsMust(t, out.Reset())
	paramsMust(t, s.SourceDisplayParameterInto(out, debugOut, "namespace.enabled"))
	if got := pdStreamText(t, debugOut); got != "namespace.enabled namespace enabled\n" {
		t.Fatal("source unsolicited parameter diagnostic", got)
	}
	parameterWant := "  name:           enabled\n  module:         namespace\n  qualified name: namespace.enabled\n  type:           bool\n  description:    " + text + "\n  default value:  false\n"
	if got := pdStreamText(t, out); got != parameterWant {
		t.Fatal("literal detailed parameter output", got, parameterWant)
	}
	paramsMust(t, s.SourceReset())
	enabled, e = s.Context.GetBoolName(&snapshot, &enabledName, false)
	paramsMust(t, e)
	if !enabled {
		t.Fatal("source module dealloc erased caller-owned returned-by-value reference")
	}
	fresh, e := s.SourceGetModuleSymbol(key)
	paramsMust(t, e)
	empty, e := s.Context.Empty(&fresh)
	paramsMust(t, e)
	if !empty {
		t.Fatal("source Reset kept module dictionary publication")
	}
	paramsMust(t, s.Context.Del(&fresh))
	paramsMust(t, s.Context.Del(&snapshot))
	gparamsServiceAudit(t, s, b)
}

func TestGlobalParamsNamespaceDisplayEveryActualOutputReservation(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			s, b := gparamsNamespaceFixture(t)
			paramsMust(t, s.InitializeSource())
			text := namespaceModule(t, s, b)
			// Source catalog registration is a real prerequisite, completed before
			// this independent output-only frontier. Namespace adds no fake event.
			g, e := s.Manager()
			paramsMust(t, e)
			paramsMust(t, g.ensureRegistered())
			outputBudget := arena.NewBudget(1 << 24)
			out, e := NewStringBuffer64(StringBufferConfig{Budget: outputBudget})
			paramsMust(t, e)
			defer func() {
				paramsMust(t, out.Close())
				if outputBudget.Used() != 0 {
					t.Error("actual output owner")
				}
			}()
			outputBudget.FailAfter(ordinal)
			e = s.SourceDisplayModuleInto(out, "namespace")
			bodyErr := e
			if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal("actual reached output refusal", e)
			}
			stats := out.Stats()
			if outputBudget.Used() != stats.OwnerBytes+stats.OverflowBytes {
				t.Fatal("exact output header/backing graph", stats, outputBudget.Used())
			}
			gparamsServiceAudit(t, s, b)
			want := "[module] namespace\n    enabled (bool) " + text + " (default: false)\n"
			bytes := make([]byte, out.Position())
			for i := range bytes {
				bytes[i], e = out.Read(uint64(i))
				paramsMust(t, e)
			}
			if !strings.HasPrefix(want, string(bytes)) {
				t.Fatal("source emitted prefix diverged", string(bytes), want)
			}
			if bodyErr == nil {
				if string(bytes) != want {
					t.Fatal("successful body emitted only prefix", string(bytes), want)
				}
				success = true
			} else if len(bytes) == len(want) {
				t.Fatal("failure did not preserve actual stopping prefix")
			}
			outputBudget.FailAfter(-1)
		})
		if success {
			t.Logf("actual output first success %d", ordinal)
			break
		}
	}
	if !success {
		t.Fatal("every actual output reservation did not reach success")
	}
}

func TestGlobalParamsNamespaceActualSymbolConstructionRefusalBeforeNull(t *testing.T) {
	s, b := gparamsNamespaceFixture(t)
	sy := s.Symbols
	before := sy.Stats()
	cold := sy.Budget.Used()
	sy.Budget.FailAfter(0)
	_, e := s.SourceGetModuleName(strings.Repeat("actual-source-symbol-", 64))
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("actual symbol constructor was replaced by namespace assertion", e)
	}
	if sy.Stats() != before || sy.Budget.Used() != cold {
		t.Fatal("first-refusal named symbol storage prefix", before, sy.Stats())
	}
	if e = s.SourceDisplayModuleInto(nil, strings.Repeat("actual-source-symbol-", 64)); !namespaceNullError(e) {
		t.Fatal("display namespace assertion did not precede symbol constructor", e)
	}
	if sy.Stats() != before || sy.Budget.Used() != cold {
		t.Fatal("unused symbol argument touched storage")
	}
	gparamsServiceAudit(t, s, b)
	sy.Budget.FailAfter(-1)
}
func TestGlobalParamsNamespaceClosedDebugReadAfterSourceModuleArgument(t *testing.T) {
	s, b := gparamsNamespaceFixture(t)
	d, _, _ := debugFixture(t)
	paramsMust(t, s.BindNamespaceAssertions(d))
	paramsMust(t, d.Close())
	before := s.Symbols.Stats().Symbols
	_, e := s.SourceGetModuleName("source-symbol-before-closed-assertions")
	if GlobalParamsSourceDebug {
		if !errors.Is(e, arena.ErrClosed) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, ErrGlobalParamsUninitialized) {
		t.Fatal(e)
	}
	if s.Symbols.Stats().Symbols != before+1 {
		t.Fatal("closed debug binding moved before actual source symbol construction")
	}
	before = s.Symbols.Stats().Symbols
	e = s.SourceDisplayModuleInto(nil, "source-symbol-after-closed-assertions")
	if GlobalParamsSourceDebug {
		if !errors.Is(e, arena.ErrClosed) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, ErrGlobalParamsUninitialized) {
		t.Fatal(e)
	}
	if s.Symbols.Stats().Symbols != before {
		t.Fatal("source display symbol constructed before actual assertion read")
	}
	gparamsServiceAudit(t, s, b)
}
