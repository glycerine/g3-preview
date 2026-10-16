//go:build g3debug && g3trace

package tactic

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: src/tactic/extension_model_converter.cpp:66;
// src/model/model_core.cpp:29; src/util/memory_manager.cpp:202-211.
func TestExtensionModelFunctionMemoryWrapperFollowsModelCore(t *testing.T) {
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, md, out, config := extensionFixture(t, tags)
	c, err := NewExtensionModel(m, config)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, c.Close()) })
	b, err := m.MkBoolSort()
	satOK(t, err)
	d, err := m.MkFuncDecl(astTestSymbol(t, m, "extension-memory-function"), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	satOK(t, err)
	v, err := m.MkVar(0, b)
	satOK(t, err)
	astTestPin(t, m, v)
	satOK(t, c.Insert(d, v))
	tags.Enable("memory")
	satOK(t, c.ApplyModel(out, 0))
	fi, err := md.Core().FunctionInterpretation(d)
	satOK(t, err)
	if fi == nil {
		t.Fatal("extension function was not published")
	}
	id := fmt.Sprintf("%p", fi)
	size := strconv.FormatUint(uint64(reflect.TypeOf(*fi).Size()), 10)
	allocation := "alloc " + id + " src/tactic/extension_model_converter.cpp:66 func_interp " + size + "\n"
	deallocation := "dealloc " + id + " src/model/model_core.cpp:29\n"
	if got := output.String(); strings.Count(got, allocation) != 1 || strings.Contains(got, deallocation) {
		t.Fatalf("extension function allocation: %q", got)
	}
	satOK(t, out.Set(nil))
	got := output.String()
	if strings.Count(got, allocation) != 1 || strings.Count(got, deallocation) != 1 || strings.Index(got, allocation) > strings.Index(got, deallocation) {
		t.Fatalf("extension function lifetime: %q", got)
	}
}
