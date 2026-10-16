package smtcore

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestParamDatatypeFixMissingRefsSourceOrderAndOverwrite(t *testing.T) {
	a, b, unknown := usymbol.Numerical(51), usymbol.Numerical(52), usymbol.Numerical(53)
	group := ParamDatatypeGroup{Datatypes: []ParamDatatypeDecl{
		{Name: a, Constructors: []ParamDatatypeConstructor{{Accessors: []ParamDatatypeAccessor{
			{SortKind: ParamMissingSort, MissingSort: b},
			{SortKind: ParamSort},
			{SortKind: ParamMissingSort, MissingSort: unknown},
			{SortKind: ParamMissingSort, MissingSort: a},
		}}}},
		{Name: b},
		{Name: a}, // source dictionary::insert replaces a's index with the last one.
	}}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	group.trace = tags
	tags.Enable("fix_missing_refs")
	missing, ok := group.FixMissingRefs()
	if ok || missing != unknown {
		t.Fatal("did not stop at first missing accessor", missing, ok)
	}
	as := group.Datatypes[0].Constructors[0].Accessors
	if as[0].SortKind != ParamRecursiveSort || as[0].RecursiveIndex != 1 || as[0].MissingSort != (usymbol.Symbol{}) {
		t.Fatal("completed accessor was not rewritten before refusal", as[0])
	}
	if as[1].SortKind != ParamSort || as[2].SortKind != ParamMissingSort || as[3].SortKind != ParamMissingSort {
		t.Fatal("refusal rewrote untouched accessor types", as)
	}
	group.Datatypes = append(group.Datatypes, ParamDatatypeDecl{Name: unknown})
	missing, ok = group.FixMissingRefs()
	if !ok || missing != (usymbol.Symbol{}) {
		t.Fatal("second pass did not resolve newly declared datatype", missing, ok)
	}
	as = group.Datatypes[0].Constructors[0].Accessors
	if as[2].SortKind != ParamRecursiveSort || as[2].RecursiveIndex != 3 || as[3].SortKind != ParamRecursiveSort || as[3].RecursiveIndex != 2 {
		t.Fatal("recursive indices did not use declaration positions", as)
	}
	if tags.ShouldTrace("fix_missing_refs") {
		const record = "-------- [fix_missing_refs] fix_missing_refs src/cmd_context/pdecl.cpp:667 ---------\n" +
			"pdatatypes_decl::fix_missing_refs\n" +
			"------------------------------------------------\n"
		if got := output.String(); got != record+record {
			t.Fatalf("source pre-resolution TRACE on failed and completed pass: got %q", got)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased datatype TRACE emitted", output.String())
	}
}

func TestParamDatatypeValidationFirstMissingAndDuplicateAcrossConstructors(t *testing.T) {
	a, b, c := usymbol.Numerical(61), usymbol.Numerical(62), usymbol.Numerical(63)
	d := ParamDatatypeDecl{Constructors: []ParamDatatypeConstructor{
		{Accessors: []ParamDatatypeAccessor{
			{Name: a, SortKind: ParamMissingSort, MissingSort: b},
			{Name: b, SortKind: ParamMissingSort, MissingSort: c},
		}},
		{Accessors: []ParamDatatypeAccessor{{Name: a, SortKind: ParamSort}}},
	}}
	if missing, found := d.HasMissingRef(); !found || missing != b {
		t.Fatal("first missing accessor", missing, found)
	}
	if repeated, found := d.HasDuplicateAccessors(); !found || repeated != a {
		t.Fatal("duplicate across constructors", repeated, found)
	}
	d.Constructors[0].Accessors[0].SortKind = ParamRecursiveSort
	if missing, found := d.HasMissingRef(); !found || missing != c {
		t.Fatal("next missing accessor", missing, found)
	}
	d.Constructors[0].Accessors[1].SortKind = ParamRecursiveSort
	if missing, found := d.HasMissingRef(); found || missing != (usymbol.Symbol{}) {
		t.Fatal("resolved datatype still reports missing accessor", missing, found)
	}
	other := ParamDatatypeDecl{Constructors: []ParamDatatypeConstructor{{Accessors: []ParamDatatypeAccessor{{Name: a, SortKind: ParamSort}}}}}
	if repeated, found := other.HasDuplicateAccessors(); found || repeated != (usymbol.Symbol{}) {
		t.Fatal("accessor namespace leaked across datatypes", repeated, found)
	}
}
