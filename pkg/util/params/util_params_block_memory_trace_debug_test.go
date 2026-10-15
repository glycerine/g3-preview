//go:build g3debug && g3trace

package params

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestParamsBlockMemoryTraceFirstCopyOnWriteAndStableIdentity(t *testing.T) {
	c, sy, _ := paramsTestOwner(t)
	var out bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&out)
	tags.Enable("memory")
	paramsMust(t, c.BindMemoryTrace(tags))
	firstKey := internSymbol(t, sy, "first-memory-block")
	secondKey := internSymbol(t, sy, "second-memory-block")
	a, e := c.NewValue()
	paramsMust(t, e)
	b, e := c.NewValue()
	paramsMust(t, e)
	paramsMust(t, c.SetBool(&a, firstKey, true))
	oldID := c.blockMemoryIdentity(a.Slot)
	oldAddress := &c.Blocks[a.Slot-1]
	paramsMust(t, c.Assign(&b, &a))
	paramsMust(t, c.SetBool(&b, secondKey, true))
	newID := c.blockMemoryIdentity(b.Slot)
	if oldID == newID || strings.Count(out.String(), " src/util/params.cpp:605 params ") != 1 {
		t.Fatal("copy-on-write did not acquire one new source block", oldID, newID, out.String())
	}
	var extras []ParamsRef
	for i := 0; i < 8; i++ {
		v, err := c.NewValue()
		paramsMust(t, err)
		paramsMust(t, c.SetBool(&v, firstKey, true))
		extras = append(extras, v)
	}
	if oldAddress == &c.Blocks[a.Slot-1] {
		t.Fatal("fixture did not relocate the Go block directory")
	}
	paramsMust(t, c.Del(&a))
	paramsMust(t, c.Del(&b))
	for i := range extras {
		paramsMust(t, c.Del(&extras[i]))
	}
	got := out.String()
	for _, check := range []struct {
		id   string
		line int
	}{{oldID, 600}, {newID, 605}} {
		allocation := fmt.Sprintf("alloc %s src/util/params.cpp:%d params %d\n", check.id, check.line, paramsBlockBytes(1))
		deallocation := fmt.Sprintf("dealloc %s src/util/params.cpp:339\n", check.id)
		if strings.Count(got, allocation) != 1 || strings.Count(got, deallocation) != 1 || strings.Index(got, deallocation) <= strings.Index(got, allocation) {
			t.Fatalf("source block trace did not retain %s through directory growth: alloc %d dealloc %d", check.id, strings.Count(got, allocation), strings.Count(got, deallocation))
		}
	}
}
