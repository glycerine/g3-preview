package rewriter

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func enum2BVQuantifier(t *testing.T, m *ast.ASTManager, sort ast.ASTHandle, n int, forall bool, body ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	sorts := make([]ast.ASTHandle, n)
	names := make([]usymbol.Symbol, n)
	for i := range sorts {
		sorts[i] = sort
		names[i] = astTestSymbol(t, m, fmt.Sprintf("enum-q%d", i))
	}
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: forall, Sorts: sorts, Names: names, Body: body, Weight: 7, QID: astTestSymbol(t, m, "enum-qid"), SkolemID: astTestSymbol(t, m, "enum-skid")})
	astTestMust(t, err)
	astTestPin(t, m, q)
	return q
}
func enum2BVQuantBody(t *testing.T, m *ast.ASTManager) ast.ASTHandle {
	t.Helper()
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	body, err := m.MkConst(astTestSymbol(t, m, "enum-quant-body"), boolean)
	astTestMust(t, err)
	astTestPin(t, m, body)
	return body
}
func enum2BVQuantProof(t *testing.T, m *ast.ASTManager, body ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	astTestMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	p, err := m.MkProofAsserted(body)
	astTestMust(t, err)
	astTestPin(t, m, p)
	return p
}
func enum2BVBound(t *testing.T, m *ast.ASTManager, index uint32) ast.ASTHandle {
	t.Helper()
	sort, err := m.BitVectorSort(2)
	astTestMust(t, err)
	variable, err := m.MkVar(index, sort)
	astTestMust(t, err)
	upper, err := m.BitVectorNumeralUnsigned(2, 2)
	astTestMust(t, err)
	bound, err := m.MkBitVectorULE(variable, upper)
	astTestMust(t, err)
	return bound
}

func TestEnum2BVQuantifierBoundsOwnedBeforeNextPredicate(t *testing.T) {
	m, sort, _ := enum2BVFixture(t)
	r, err := enum2BVNewForTest(t, m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	body := enum2BVQuantBody(t, m)
	q := enum2BVQuantifier(t, m, sort, 2, true, body)
	bound := enum2BVBound(t, m, 1)
	astTestPin(t, m, bound)
	before, err := m.GetRefCount(bound)
	astTestMust(t, err)
	variable, err := m.GetArg(bound, 0)
	astTestMust(t, err)
	bvSort, err := m.GetSort(variable)
	astTestMust(t, err)
	calls := 0
	stop := errors.New("next binder predicate stopped")
	astTestMust(t, r.SetSortPredicate(func(ast.ASTHandle) (bool, error) {
		calls++
		if calls == 2 {
			count, err := m.GetRefCount(bound)
			if err != nil {
				return false, err
			}
			if count != before+1 || len(r.config.sorts) != 1 || r.config.sorts[0] != bvSort {
				t.Fatal("bound/sort prefix not published before next source predicate", count, before, len(r.config.sorts))
			}
			return false, stop
		}
		return true, nil
	}))
	out, proof := enum2BVRefs(t, m)
	_, err = r.config.ReduceQuantifier(q, body, ast.SliceArguments(nil), ast.SliceArguments(nil), out, proof)
	if !errors.Is(err, stop) || calls != 2 || len(r.pending) != 0 {
		t.Fatal("binder failure lost its source prefix", err, calls)
	}
	after, err := m.GetRefCount(bound)
	astTestMust(t, err)
	if after != before || len(r.config.sorts) != 1 {
		t.Fatal("bound owner survived failure or member sort prefix disappeared", after, before)
	}
}

func TestEnum2BVQuantifierOwnersMetadataAndProofClear(t *testing.T) {
	for _, forall := range []bool{false, true} {
		t.Run(fmt.Sprint(forall), func(t *testing.T) {
			m, sort, _ := enum2BVFixture(t)
			body := enum2BVQuantBody(t, m)
			oldProof := enum2BVQuantProof(t, m, body)
			r, err := enum2BVNewForTest(t, m)
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, r.Close()) })
			q := enum2BVQuantifier(t, m, sort, 1, forall, body)
			_, err = r.utility.IsEnumSort(sort)
			astTestMust(t, err)
			out := astTestPin(t, m, q)
			proof := astTestPin(t, m, oldProof)
			before := m.Stats().LiveReferences
			changed, err := r.config.ReduceQuantifier(q, body, ast.SliceArguments(nil), ast.SliceArguments(nil), out, proof)
			astTestMust(t, err)
			result, err := m.RefGet(out)
			astTestMust(t, err)
			cleared, err := m.RefGet(proof)
			astTestMust(t, err)
			if !changed || result == q || !cleared.IsNull() || m.Stats().LiveReferences != before {
				t.Fatal("quantifier output/proof/local retirement mismatch", changed, result, cleared, m.Stats().LiveReferences, before)
			}
			rewritten, err := m.GetQuantifierBody(result)
			astTestMust(t, err)
			kind, err := m.GetDeclKind(rewritten)
			astTestMust(t, err)
			want := ast.ASTOpAnd
			if forall {
				want = ast.ASTOpImplies
			}
			if kind != want {
				t.Fatal("wrong source quantifier guard", kind, want)
			}
			weight, err := m.GetQuantifierWeight(result)
			astTestMust(t, err)
			qid, skid, err := m.GetQuantifierIDs(result)
			astTestMust(t, err)
			oldQid, oldSkid, err := m.GetQuantifierIDs(q)
			astTestMust(t, err)
			if weight != 7 || qid != oldQid || skid != oldSkid {
				t.Fatal("quantifier metadata changed")
			}
			name, err := m.GetDeclarationName(result, 0)
			astTestMust(t, err)
			oldName, err := m.GetDeclarationName(q, 0)
			astTestMust(t, err)
			if name != oldName {
				t.Fatal("quantifier declaration name changed")
			}
			// A later non-enum call resets m_sorts but does not touch either output.
			boolean, err := m.MkBoolSort()
			astTestMust(t, err)
			ordinary := enum2BVQuantifier(t, m, boolean, 1, forall, body)
			astTestMust(t, m.SetRef(proof, oldProof))
			enum2BVFillReferencePage(t, m, body, 0)
			m.Budget().FailAfter(0)
			changed, err = r.config.ReduceQuantifier(ordinary, body, ast.SliceArguments(nil), ast.SliceArguments(nil), out, proof)
			m.Budget().FailAfter(-1)
			astTestMust(t, err)
			unchanged, err := m.RefGet(out)
			astTestMust(t, err)
			retainedProof, err := m.RefGet(proof)
			astTestMust(t, err)
			if changed || unchanged != result || retainedProof != oldProof || len(r.config.sorts) != 1 || r.config.sorts[0] != boolean {
				t.Fatal("no-enum path changed outputs or retained old sort buffer")
			}
			astTestMust(t, r.Close())
			if r.config.sorts != nil {
				t.Fatal("Close retained the destroyed source member sort buffer")
			}
		})
	}
}

func TestEnum2BVQuantifierNoBoundsStillOwnsBody(t *testing.T) {
	m, _, _ := enum2BVFixture(t)
	sorts, err := m.MkDatatypes([]ast.DatatypeDefinition{{Name: astTestSymbol(t, m, "BinaryEnum"), Constructors: []ast.DatatypeConstructorDefinition{{Name: astTestSymbol(t, m, "be0"), Recognizer: astTestSymbol(t, m, "is-be0")}, {Name: astTestSymbol(t, m, "be1"), Recognizer: astTestSymbol(t, m, "is-be1")}}}})
	astTestMust(t, err)
	sort := sorts[0]
	body := enum2BVQuantBody(t, m)
	oldProof := enum2BVQuantProof(t, m, body)
	r, err := enum2BVNewForTest(t, m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	q := enum2BVQuantifier(t, m, sort, 1, true, body)
	bv, err := m.BitVectorSort(1)
	astTestMust(t, err)
	// Canonicalize the final result too. The census below distinguishes body
	// reference-page growth from quantifier payload allocation refusals.
	_ = enum2BVQuantifier(t, m, bv, 1, true, body)
	_, err = r.utility.IsEnumSort(sort)
	astTestMust(t, err)
	out := astTestPin(t, m, q)
	proof := astTestPin(t, m, oldProof)
	enum2BVFillReferencePage(t, m, body, 0)
	m.Budget().FailAfter(0)
	_, err = r.config.ReduceQuantifier(q, body, ast.SliceArguments(nil), ast.SliceArguments(nil), out, proof)
	m.Budget().FailAfter(-1)
	if !errors.Is(err, arena.ErrInjectedFailure) || len(r.config.sorts) != 1 || r.config.sorts[0] != bv {
		t.Fatal("no-bounds body reference reservation was skipped or refused before source stage", err, len(r.config.sorts))
	}
	unchanged, err := m.RefGet(out)
	astTestMust(t, err)
	old, err := m.RefGet(proof)
	astTestMust(t, err)
	if unchanged != q || old != oldProof || len(r.pending) != 0 {
		t.Fatal("body pin failure changed caller outputs")
	}
	// Generic allocation errors can also come from quantifier payloads.
	// Census until the body reference page grows, then prove the retired
	// slot is reusable with allocation disabled. Canonical construction may
	// finish before another refusal in native builds.
	observedOwner := false
	for ordinal := int64(1); ordinal < 32; ordinal++ {
		m.Budget().FailAfter(ordinal)
		changed, callErr := r.config.ReduceQuantifier(q, body, ast.SliceArguments(nil), ast.SliceArguments(nil), out, proof)
		m.Budget().FailAfter(-1)
		if callErr != nil && !errors.Is(callErr, arena.ErrInjectedFailure) {
			t.Fatal("unexpected body ownership census refusal", ordinal, callErr)
		}
		m.Budget().FailAfter(0)
		probe, probeErr := m.NewRef(body)
		m.Budget().FailAfter(-1)
		if probeErr == nil {
			astTestMust(t, m.DelRef(&probe))
			observedOwner = true
			value, err := m.RefGet(out)
			astTestMust(t, err)
			retained, err := m.RefGet(proof)
			astTestMust(t, err)
			if callErr != nil {
				if value != q || retained != oldProof {
					t.Fatal("later construction refusal changed outputs")
				}
			} else if !changed || value == q || !retained.IsNull() {
				t.Fatal("successful no-bounds rewrite lost its output/proof prefix")
			}
			break
		}
		if !errors.Is(probeErr, arena.ErrInjectedFailure) {
			t.Fatal(probeErr)
		}
		if callErr == nil {
			break
		}
	}
	if !observedOwner {
		t.Fatal("no body owner was acquired and retired during quantifier construction")
	}

}

func TestEnum2BVQuantifierSourceLexicalRetirementOrder(t *testing.T) {
	s, q, b := astTestOwners(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: q, Budget: b, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	astTestMust(t, m.RegisterDeclarationPlugins())
	scope := &enum2BVQuantifierScope{manager: m}
	body := func() ast.ASTHandle {
		sort, err := m.MkBoolSort()
		astTestMust(t, err)
		body, err := m.MkConst(astTestSymbol(t, m, "retired-quantifier-body"), sort)
		astTestMust(t, err)
		return body
	}()
	bound0, bound1 := enum2BVBound(t, m, 0), enum2BVBound(t, m, 1)
	astTestMust(t, scope.push(bound0))
	astTestMust(t, scope.push(bound1))
	astTestMust(t, enum2BVAssign(m, &scope.body, body))
	ids := make([]uint32, 3)
	for i, h := range []ast.ASTHandle{body, bound0, bound1} {
		ids[i], err = m.GetASTID(h)
		astTestMust(t, err)
	}
	tags.Enable("ast")
	astTestMust(t, scope.close())
	if !tags.ShouldTrace("ast") {
		return
	}
	last := -1
	for _, id := range ids {
		at := strings.Index(output.String(), fmt.Sprintf("Deleting object %d ", id))
		if at <= last {
			t.Fatal("source quantifier destruction is not body-before-forward-bounds", ids, output.String())
		}
		last = at
	}
}

func TestEnum2BVQuantifierStoppedCleanupBeforeMembers(t *testing.T) {
	s, q, b := astTestOwners(t)
	writer := &enum2BVStopWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: q, Budget: b, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	astTestMust(t, m.RegisterDeclarationPlugins())
	r, err := enum2BVNewForTest(t, m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	body := enum2BVQuantBody(t, m)
	bound := enum2BVBound(t, m, 0)
	astTestPin(t, m, bound)
	beforeBody, err := m.GetRefCount(body)
	astTestMust(t, err)
	beforeBound, err := m.GetRefCount(bound)
	astTestMust(t, err)
	scope := &enum2BVQuantifierScope{manager: m}
	astTestMust(t, scope.push(bound))
	astTestMust(t, enum2BVAssign(m, &scope.body, body))
	fp, err := m.FloatingPointSort(3, 4)
	astTestMust(t, err)
	writer.action = func() error { var err error; r.finishQuantifier(scope, &err); return errors.Join(err, r.Close()) }
	tags.Enable("ast")
	if !tags.ShouldTrace("ast") {
		astTestMust(t, scope.close())
		t.Skip("requires source trace build")
	}
	_, err = m.MkFPAZeroOfSort(fp, false)
	astTestMust(t, err)
	if !writer.called || !errors.Is(writer.err, ast.ErrASTPending) || len(r.pending) != 1 || r.pending[0].quantifier != scope {
		t.Fatal("stopped quantifier owner was not retained before members", writer.err, len(r.pending))
	}
	afterBody, err := m.GetRefCount(body)
	astTestMust(t, err)
	afterBound, err := m.GetRefCount(bound)
	astTestMust(t, err)
	if afterBody != beforeBody+1 || afterBound != beforeBound+1 {
		t.Fatal("unconsumed quantifier cleanup lost pins")
	}
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	if _, err := r.config.utility.IsEnumSort(boolean); err != nil {
		t.Fatal("quantifier cleanup crossed member barrier", err)
	}
	astTestMust(t, r.Close())
	astTestMust(t, r.Close())
	afterBody, err = m.GetRefCount(body)
	astTestMust(t, err)
	afterBound, err = m.GetRefCount(bound)
	astTestMust(t, err)
	if afterBody != beforeBody || afterBound != beforeBound || len(r.pending) != 0 {
		t.Fatal("quantifier retry retained/replayed pins")
	}
}
