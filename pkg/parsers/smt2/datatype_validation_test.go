package smt2

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/smtcore"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestSMT2DatatypeNameChecksSourcePriorityAndPartialRewrite(t *testing.T) {
	budget := arena.NewBudget(1 << 20)
	symbols, err := usymbol.NewManager(usymbol.Config{Budget: budget})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := symbols.Close(); err != nil {
			t.Fatal(err)
		}
		if budget.Used() != 0 {
			t.Fatal("symbol owner retained memory", budget.Used())
		}
	})
	name := func(s string) usymbol.Symbol {
		t.Helper()
		v, err := symbols.Intern(s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	a, b, unknown, field := name("A"), name("B"), name("Unknown"), name("field")
	check := func(err error, message string) {
		t.Helper()
		var ex smtcore.CmdException
		if !errors.As(err, &ex) || ex.Error() != message || !ex.HasPos() {
			t.Fatal("parser diagnostic", err)
		}
		line, lineErr := ex.Line()
		pos, posErr := ex.Pos()
		if lineErr != nil || posErr != nil || line != 7 || pos != 13 {
			t.Fatal("parser diagnostic position", line, pos, lineErr, posErr)
		}
	}
	single := smtcore.ParamDatatypeGroup{Datatypes: []smtcore.ParamDatatypeDecl{{Name: a, Constructors: []smtcore.ParamDatatypeConstructor{{Accessors: []smtcore.ParamDatatypeAccessor{
		{Name: field, SortKind: smtcore.ParamMissingSort, MissingSort: unknown},
		{Name: field, SortKind: smtcore.ParamSort},
	}}}}}}
	check(validateDatatypeNames(symbols, &single, 7, 13), "invalid datatype declaration, unknown sort 'Unknown'")
	if single.Datatypes[0].Constructors[0].Accessors[0].SortKind != smtcore.ParamMissingSort {
		t.Fatal("single declaration rewrote an unknown sort")
	}
	multi := smtcore.ParamDatatypeGroup{Datatypes: []smtcore.ParamDatatypeDecl{
		{Name: a, Constructors: []smtcore.ParamDatatypeConstructor{{Accessors: []smtcore.ParamDatatypeAccessor{
			{Name: field, SortKind: smtcore.ParamMissingSort, MissingSort: b},
			{Name: field, SortKind: smtcore.ParamSort},
		}}}},
		{Name: b, Constructors: []smtcore.ParamDatatypeConstructor{{Accessors: []smtcore.ParamDatatypeAccessor{{Name: name("other"), SortKind: smtcore.ParamMissingSort, MissingSort: a}}}}},
	}}
	check(validateDatatypeNames(symbols, &multi, 7, 13), "invalid datatype declaration, repeated accessor identifier 'field'")
	if got := multi.Datatypes[0].Constructors[0].Accessors[0]; got.SortKind != smtcore.ParamRecursiveSort || got.RecursiveIndex != 1 {
		t.Fatal("group reference fix did not precede duplicate check", got)
	}
	if got := multi.Datatypes[1].Constructors[0].Accessors[0]; got.SortKind != smtcore.ParamRecursiveSort || got.RecursiveIndex != 0 {
		t.Fatal("later group reference was not fixed", got)
	}
	multi.Datatypes[0].Constructors[0].Accessors[1].Name = name("unique")
	if err := validateDatatypeNames(symbols, &multi, 7, 13); err != nil {
		t.Fatal("valid group", err)
	}
	unknownGroup := smtcore.ParamDatatypeGroup{Datatypes: []smtcore.ParamDatatypeDecl{
		{Name: a, Constructors: []smtcore.ParamDatatypeConstructor{{Accessors: []smtcore.ParamDatatypeAccessor{
			{Name: field, SortKind: smtcore.ParamMissingSort, MissingSort: b},
			{Name: field, SortKind: smtcore.ParamMissingSort, MissingSort: unknown},
		}}}},
		{Name: b},
	}}
	check(validateDatatypeNames(symbols, &unknownGroup, 7, 13), "invalid datatype declaration, unknown sort 'Unknown'")
	as := unknownGroup.Datatypes[0].Constructors[0].Accessors
	if as[0].SortKind != smtcore.ParamRecursiveSort || as[1].SortKind != smtcore.ParamMissingSort {
		t.Fatal("failed group did not retain source prefix rewrite", as)
	}
}
