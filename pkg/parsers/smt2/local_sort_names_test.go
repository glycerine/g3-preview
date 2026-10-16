package smt2

import (
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMT2SortAndDatatypeNameTablesSourceScopes(t *testing.T) {
	var names smt2SortNames
	a := usymbol.Numerical(41)
	b := usymbol.Numerical(42)
	names.beginSortParams()
	names.addSortParam(a, 0)
	names.addSortParam(b, 1)
	names.addSortParam(a, 2) // map::insert overwrites an equal key, but size stays two.
	if size, index, found := names.sortParam(a); !found || size != 2 || index != 2 {
		t.Fatal("sort parameter overwrite", size, index, found)
	}
	if size, _, found := names.sortParam(usymbol.Numerical(43)); found || size != 2 {
		t.Fatal("unknown sort parameter", size, found)
	}
	names.beginDatatypeDecls()
	if size, _, found := names.sortParam(a); found || size != 0 {
		t.Fatal("datatype declaration retained earlier sort parameters", size, found)
	}
	names.addDatatype(a, 0, 1)
	names.addDatatype(b, 1, 3)
	names.addDatatype(a, 2, 4)
	if index, found := names.lookupDatatypeIndex(a); !found || index != 2 {
		t.Fatal("datatype index overwrite", index, found)
	}
	if arity, found := names.lookupDatatypeArity(a); !found || arity != 4 {
		t.Fatal("datatype arity overwrite", arity, found)
	}
	if len(names.datatypeNames) != 3 || names.datatypeNames[0] != a || names.datatypeNames[1] != b || names.datatypeNames[2] != a {
		t.Fatal("datatype declaration order differs from source vector")
	}
	names.beginDatatypeBodies()
	if _, found := names.lookupDatatypeIndex(a); found {
		t.Fatal("datatype body retained its declaration index")
	}
	if arity, found := names.lookupDatatypeArity(a); !found || arity != 4 {
		t.Fatal("datatype body lost its declaration arity", arity, found)
	}
	names.addSortParam(b, 5)
	names.addDatatype(b, 6, 7)
	names.reset()
	if size, _, found := names.sortParam(b); found || size != 0 {
		t.Fatal("parser reset retained sort parameter", size, found)
	}
	if _, found := names.lookupDatatypeIndex(b); found {
		t.Fatal("parser reset retained datatype index")
	}
	if arity, found := names.lookupDatatypeArity(b); !found || arity != 7 || len(names.datatypeNames) != 4 {
		t.Fatal("parser reset changed deferred datatype metadata", arity, found)
	}
	names.beginDatatypeDecls()
	if _, found := names.lookupDatatypeArity(b); found || len(names.datatypeNames) != 0 {
		t.Fatal("new datatype declaration retained prior metadata")
	}
}
