// Source: src/util/memory_manager.cpp:103; scripts/mk_genfile_common.py:742-802;
// src/util/gparams.cpp:557-631; src/util/debug.h:63; src/util/symbol.h:96.
// Source authors: Nikolaj Bjorner; Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestMemoryProcessPreparedNamespaceBorrowsActualAssertionOwner(t *testing.T) {
	p, _ := memoryProcessFixture(t)
	paramsMust(t, p.config.Debug.EnableAssertions(false))
	paramsMust(t, p.Initialize(0))
	g, e := p.parameters.Manager()
	paramsMust(t, e)
	_, want := g.GetValueName("")
	_, got := p.parameters.SourceGetValueSymbol(NumericalSymbol(1))
	if want == nil || got == nil || want.Error() != got.Error() {
		t.Fatal("disabled process assertion must reach actual numeric bare_str empty-name engine", want, got)
	}
	memoryProcessColdAudit(t, p)
	paramsMust(t, p.config.Debug.EnableAssertions(true))
	_, got = p.parameters.SourceGetValueSymbol(NumericalSymbol(1))
	if GlobalParamsSourceDebug && !errors.Is(got, ErrGlobalParamsAssertion) {
		t.Fatal("same borrowed owner enable must affect reached bare_str assertion", got)
	}
}

func TestMemoryProcessPreparedNamespaceNullGuardUsesCurrentAssertionState(t *testing.T) {
	p, _ := memoryProcessFixture(t)
	paramsMust(t, p.Initialize(0))
	paramsMust(t, p.parameters.FinalizeSource())
	paramsMust(t, p.config.Debug.EnableAssertions(false))
	before := p.config.Runtime.State()
	if e := p.parameters.SourceReset(); !errors.Is(e, ErrGlobalParamsUninitialized) {
		t.Fatal("disabled source SASSERT must precede reached typed null-imp guard", e)
	}
	memoryProcessColdAudit(t, p)
	paramsMust(t, p.config.Debug.EnableAssertions(true))
	e := p.parameters.SourceReset()
	if GlobalParamsSourceDebug {
		if !errors.Is(e, ErrGlobalParamsAssertion) {
			t.Fatal("enabled source null-imp SASSERT", e)
		}
	} else if !errors.Is(e, ErrGlobalParamsUninitialized) {
		t.Fatal("release erased assertion", e)
	}
	if p.config.Runtime.State() != before {
		t.Fatal("null-imp assertion query changed Runtime")
	}
}

func TestMemoryProcessPreparedNamespaceReadsBorrowedDebugAtActualSite(t *testing.T) {
	p, _ := memoryProcessFixture(t)
	paramsMust(t, p.Initialize(0))
	paramsMust(t, p.parameters.FinalizeSource())
	paramsMust(t, p.config.Debug.Close())
	symbols, e := p.config.Symbols.Manager()
	paramsMust(t, e)
	before := symbols.Stats().Symbols
	_, e = p.parameters.SourceGetModuleName("process-symbol-before-closed-assertion")
	if GlobalParamsSourceDebug {
		if !errors.Is(e, arena.ErrClosed) {
			t.Fatal("reached actual borrowed debug lifetime", e)
		}
	} else if !errors.Is(e, ErrGlobalParamsUninitialized) {
		t.Fatal("release assertion borrow erased", e)
	}
	if symbols.Stats().Symbols != before+1 {
		t.Fatal("get_module symbol construction must precede assertion read")
	}
	before = symbols.Stats().Symbols
	e = p.parameters.SourceDisplayModuleInto(nil, "process-symbol-after-closed-assertion")
	if GlobalParamsSourceDebug {
		if !errors.Is(e, arena.ErrClosed) {
			t.Fatal("display reached borrowed debug", e)
		}
	} else if !errors.Is(e, ErrGlobalParamsUninitialized) {
		t.Fatal("release display null-imp guard", e)
	}
	if symbols.Stats().Symbols != before {
		t.Fatal("display assertion must precede symbol construction")
	}
	memoryProcessColdAudit(t, p)
}
