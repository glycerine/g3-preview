//go:build g3debug && g3trace

package ast

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestASTFormatManagerMemoryWrapperConstructorOverloads(t *testing.T) {
	for _, borrowed := range []bool{false, true} {
		t.Run(strconv.FormatBool(borrowed), func(t *testing.T) {
			sy, q, _ := astTestOwners(t)
			b := arena.NewBudget(1 << 28)
			var output bytes.Buffer
			tags := udiag.NewTraceTagsWithWriter(&output)
			tags.Enable("memory")
			var stream *utext.NumericTextStream
			var err error
			if borrowed {
				stream, err = utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b})
				astTestMust(t, err)
			}
			m, err := constructSourceASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b, TraceTags: tags}, false, nil, stream)
			astTestMust(t, err)
			child := m.Format
			if child == nil || !child.formatTraceLive {
				t.Fatal("missing owned format child")
			}
			astTestMust(t, m.Close())
			astTestMust(t, m.Close())
			if stream != nil {
				astTestMust(t, stream.Close())
			}
			line := 1313
			if borrowed {
				line = 1330
			}
			id := fmt.Sprintf("%p", child)
			size := strconv.FormatUint(uint64(reflect.TypeOf(ASTManager{}).Size()), 10)
			want := "-------- [memory] allocate src/util/memory_manager.cpp:209 ---------\n" +
				"alloc " + id + " src/ast/ast.cpp:" + strconv.Itoa(line) + " ast_manager " + size + "\n" +
				"------------------------------------------------\n" +
				"-------- [memory] deallocate src/util/memory_manager.cpp:204 ---------\n" +
				"dealloc " + id + " src/ast/ast.cpp:1473\n" +
				"------------------------------------------------\n"
			if got := output.String(); got != want || b.Used() != 0 || child.formatTraceLive {
				t.Fatalf("format child memory records: got %q want %q used %d live %v", got, want, b.Used(), child.formatTraceLive)
			}
		})
	}
}
