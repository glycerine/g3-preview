//go:build g3debug && g3trace

package model_test

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestModelCopyMemoryWrapperSuccessfulAndRefusedHeader(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	original, err := model.NewModel(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, original.Close()) })
	tags.Enable("memory")
	copied, err := original.Copy()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, copied.Close()) })
	id := fmt.Sprintf("%p", copied)
	size := strconv.FormatUint(uint64(reflect.TypeOf(model.Model{}).Size()), 10)
	alloc := "alloc " + id + " src/model/model.cpp:69 model " + size + "\n"
	dealloc := "dealloc " + id + " src/model/model_core.h:74\n"
	if strings.Count(output.String(), alloc) != 1 || strings.Contains(output.String(), dealloc) {
		t.Fatalf("copy allocation before Close: %q", output.String())
	}
	budget.FailAfter(0)
	failed, err := original.Copy()
	budget.FailAfter(-1)
	if failed != nil || !errors.Is(err, arena.ErrInjectedFailure) {
		t.Fatalf("refused model header: %p %v", failed, err)
	}
	astTestMust(t, copied.Close())
	astTestMust(t, copied.Close())
	astTestMust(t, original.Close())
	got := output.String()
	if strings.Count(got, alloc) != 1 || strings.Count(got, dealloc) != 1 ||
		strings.Index(got, alloc) > strings.Index(got, dealloc) {
		t.Fatalf("copy lifetime records: %q", got)
	}
}
