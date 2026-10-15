//go:build g3debug && g3trace

package params

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestParamsRationalEntryMemoryTraceNewMatchedAndReleasePaths(t *testing.T) {
	c, sy, q := paramsTestOwner(t)
	var out bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&out)
	tags.Enable("memory")
	paramsMust(t, c.BindMemoryTrace(tags))
	first := internSymbol(t, sy, "first-rational-memory-key")
	second := internSymbol(t, sy, "second-rational-memory-key")
	third := internSymbol(t, sy, "third-rational-memory-key")
	r, e := q.NewValue()
	paramsMust(t, e)
	paramsMust(t, q.AssignInt(&r, 7))
	a, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetBool(&a, first, true))
	paramsMust(t, c.SetRat(&a, second, &r))
	b := &c.Blocks[a.Slot-1]
	newID := c.Entry(b, 1).ratTraceID
	paramsMust(t, c.SetRat(&a, second, &r)) // same numeral owner, no alloc
	paramsMust(t, c.SetBool(&a, second, false))
	paramsMust(t, c.SetRat(&a, second, &r)) // matched scalar allocation
	b = &c.Blocks[a.Slot-1]
	matchedID := c.Entry(b, 1).ratTraceID
	if newID == 0 || matchedID == 0 || newID == matchedID {
		t.Fatal("rational entry identities", newID, matchedID)
	}
	paramsMust(t, c.SetBool(&a, third, true)) // grow the entry page
	paramsMust(t, c.ResetKey(&a, first))      // shift the live rational entry
	b = &c.Blocks[a.Slot-1]
	if c.Entry(b, 0).ratTraceID != matchedID {
		t.Fatal("source suffix shift lost rational ownership identity")
	}
	paramsMust(t, c.ResetKey(&a, second))
	paramsMust(t, c.SetRat(&a, second, &r))
	b = &c.Blocks[a.Slot-1]
	resetID := c.Entry(b, b.Size-1).ratTraceID
	paramsMust(t, c.Reset(&a))
	paramsMust(t, c.SetRat(&a, second, &r))
	b = &c.Blocks[a.Slot-1]
	finalID := c.Entry(b, 0).ratTraceID
	paramsMust(t, c.Del(&a))
	paramsMust(t, q.Del(&r))
	got := out.String()
	for _, check := range []struct {
		id   uint64
		line int
	}{{newID, 1004}, {matchedID, 996}, {resetID, 1004}, {finalID, 1004}} {
		identity := c.rationalEntryMemoryIdentity(check.id)
		alloc := fmt.Sprintf("alloc %s src/util/params.cpp:%d rational %d\n", identity, check.line, reflect.TypeOf(unumeral.OwnedRational{}).Size())
		dealloc := fmt.Sprintf("dealloc %s src/util/params.cpp:757\n", identity)
		if check.id == 0 || strings.Count(got, alloc) != 1 || strings.Count(got, dealloc) != 1 || strings.Index(got, dealloc) <= strings.Index(got, alloc) {
			t.Fatalf("rational entry trace %s: alloc %d dealloc %d", identity, strings.Count(got, alloc), strings.Count(got, dealloc))
		}
	}
	if strings.Count(got, " rational ") != 4 {
		t.Fatal("matched numeral overwrite allocated an extra rational entry", got)
	}
}
