package api

// Public array helpers exercise the retained array declaration plugin without
// requiring callers to construct family applications directly.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTContextArrayHelpersNativeAndNested(t *testing.T) {
	f := newSMTAPIFixture(t, "ALL", nil)
	boolean := f.pin(f.m.MkBoolSort())
	falseValue := f.pin(f.m.MkBoolVal(false))
	trueValue := f.pin(f.m.MkBoolVal(true))

	nativeSort := f.pin(f.c.ArraySort([]ast.ASTHandle{boolean, boolean}, boolean))
	native := f.pin(f.c.ConstantArray(nativeSort, falseValue))
	native = f.pin(f.c.ArrayStore(native, []ast.ASTHandle{trueValue, falseValue}, trueValue))
	selected := f.pin(f.c.ArraySelect(native, trueValue, falseValue))
	selectedSort := f.pin(f.m.GetSort(selected))
	if selectedSort != boolean {
		t.Fatal("native select range sort", selectedSort, boolean)
	}

	nestedSort := f.pin(f.c.NestedArraySort([]ast.ASTHandle{boolean, boolean}, boolean))
	innerSort := f.pin(f.m.ArrayRange(nestedSort))
	if leaf := f.pin(f.m.ArrayRange(innerSort)); leaf != boolean {
		t.Fatal("nested array leaf sort", leaf, boolean)
	}
	nested := f.pin(f.c.NestedConstantArray([]ast.ASTHandle{boolean, boolean}, falseValue))
	if sort := f.pin(f.m.GetSort(nested)); sort != nestedSort {
		t.Fatal("nested constant sort", sort, nestedSort)
	}
	nested = f.pin(f.c.NestedArrayStore(nested, []ast.ASTHandle{trueValue, falseValue}, trueValue))
	selected = f.pin(f.c.NestedArraySelect(nested, trueValue, falseValue))
	selectedSort = f.pin(f.m.GetSort(selected))
	if selectedSort != boolean {
		t.Fatal("nested select range sort", selectedSort, boolean)
	}
}

func TestSMTContextArrayMapDefaultAndSetHelpers(t *testing.T) {
	f := newSMTAPIFixture(t, "ALL", nil)
	boolean := f.pin(f.m.MkBoolSort())
	falseValue := f.pin(f.m.MkFalse())
	trueValue := f.pin(f.m.MkTrue())
	arraySort := f.pin(f.c.ArraySort([]ast.ASTHandle{boolean}, boolean))
	emptyArray := f.pin(f.c.ConstantArray(arraySort, falseValue))

	not := f.pin(f.m.MkNotDecl())
	mapped := f.pin(f.c.ArrayMap(not, emptyArray))
	selected := f.pin(f.c.ArraySelect(mapped, trueValue))
	if sort := f.pin(f.m.GetSort(selected)); sort != boolean {
		t.Fatal("mapped array range", sort, boolean)
	}
	defaultValue := f.pin(f.c.ArrayDefault(emptyArray))
	if sort := f.pin(f.m.GetSort(defaultValue)); sort != boolean {
		t.Fatal("array default range", sort, boolean)
	}

	setSort := f.pin(f.c.SetSort(boolean))
	if setSort != arraySort {
		t.Fatal("set sort is not its array representation", setSort, arraySort)
	}
	empty := f.pin(f.c.EmptySet(boolean))
	full := f.pin(f.c.FullSet(boolean))
	added := f.pin(f.c.SetAdd(empty, trueValue))
	deleted := f.pin(f.c.SetDelete(full, trueValue))
	member := f.pin(f.c.SetMember(trueValue, added))
	if sort := f.pin(f.m.GetSort(member)); sort != boolean {
		t.Fatal("set membership range", sort, boolean)
	}

	operations := []struct {
		kind int32
		term ast.ASTHandle
	}{
		{ast.ASTArrayOpSetUnion, f.pin(f.c.SetUnion(empty, full))},
		{ast.ASTArrayOpSetIntersect, f.pin(f.c.SetIntersection(empty, full))},
		{ast.ASTArrayOpSetDifference, f.pin(f.c.SetDifference(full, empty))},
		{ast.ASTArrayOpSetComplement, f.pin(f.c.SetComplement(empty))},
		{ast.ASTArrayOpSetSubset, f.pin(f.c.SetSubset(empty, full))},
	}
	for _, operation := range operations {
		if kind, err := f.m.GetDeclKind(operation.term); err != nil || kind != operation.kind {
			t.Fatal("set operation", operation.kind, kind, err)
		}
	}
	if kind, err := f.m.GetDeclKind(added); err != nil || kind != ast.ASTArrayOpStore {
		t.Fatal("set add representation", kind, err)
	}
	if kind, err := f.m.GetDeclKind(deleted); err != nil || kind != ast.ASTArrayOpStore {
		t.Fatal("set delete representation", kind, err)
	}
}

func TestSMTContextArrayHelpersRejectMalformedRequests(t *testing.T) {
	f := newSMTAPIFixture(t, "ALL", nil)
	boolean := f.pin(f.m.MkBoolSort())
	value := f.pin(f.m.MkBoolVal(false))
	arraySort := f.pin(f.c.ArraySort([]ast.ASTHandle{boolean}, boolean))
	array := f.pin(f.c.ConstantArray(arraySort, value))

	checks := []struct {
		name string
		call func() error
	}{
		{"array-sort", func() error { _, err := f.c.ArraySort(nil, boolean); return err }},
		{"nested-sort", func() error { _, err := f.c.NestedArraySort(nil, boolean); return err }},
		{"nested-constant", func() error { _, err := f.c.NestedConstantArray(nil, value); return err }},
		{"select", func() error { _, err := f.c.ArraySelect(array); return err }},
		{"store", func() error { _, err := f.c.ArrayStore(array, nil, value); return err }},
		{"nested-select", func() error { _, err := f.c.NestedArraySelect(array); return err }},
		{"nested-store", func() error { _, err := f.c.NestedArrayStore(array, nil, value); return err }},
		{"array-map", func() error { _, err := f.c.ArrayMap(value); return err }},
		{"set-union", func() error { _, err := f.c.SetUnion(); return err }},
		{"set-intersection", func() error { _, err := f.c.SetIntersection(); return err }},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.call(); !errors.Is(err, ast.ErrASTDomain) {
				t.Fatal("domain guard", err)
			}
		})
	}
	if _, err := f.c.ArraySelect(array, value, value); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("index arity guard", err)
	}
	if _, err := f.c.ConstantArray(arraySort, array); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("constant range guard", err)
	}
	if _, err := f.c.ArrayStore(array, []ast.ASTHandle{value}, array); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("store range guard", err)
	}
	if _, err := f.c.SetComplement(value); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("set sort guard", err)
	}
}

func TestSMTContextArrayHelpersClosedGuards(t *testing.T) {
	c, err := NewSMTContext()
	astTestMust(t, err)
	astTestMust(t, c.Close())
	if _, err = c.ArraySort([]ast.ASTHandle{{}}, ast.ASTHandle{}); !errors.Is(err, ErrSMTContextOwner) {
		t.Fatal("closed array sort", err)
	}
	if _, err = c.NestedConstantArray([]ast.ASTHandle{{}}, ast.ASTHandle{}); !errors.Is(err, ErrSMTContextOwner) {
		t.Fatal("closed nested constant", err)
	}
	if _, err = c.ArraySelect(ast.ASTHandle{}, ast.ASTHandle{}); !errors.Is(err, ErrSMTContextOwner) {
		t.Fatal("closed array select", err)
	}
	if _, err = c.NestedArrayStore(ast.ASTHandle{}, []ast.ASTHandle{{}}, ast.ASTHandle{}); !errors.Is(err, ErrSMTContextOwner) {
		t.Fatal("closed nested store", err)
	}
	if _, err = c.ArrayDefault(ast.ASTHandle{}); !errors.Is(err, ErrSMTContextOwner) {
		t.Fatal("closed array default", err)
	}
	if _, err = c.SetSort(ast.ASTHandle{}); !errors.Is(err, ErrSMTContextOwner) {
		t.Fatal("closed set sort", err)
	}
	if _, err = c.SetAdd(ast.ASTHandle{}, ast.ASTHandle{}); !errors.Is(err, ErrSMTContextOwner) {
		t.Fatal("closed set add", err)
	}
}
