//go:build g3debug && g3trace

package diagnostic

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestTraceTagTableAllocationFollowsSourceGate(t *testing.T) {
	var out bytes.Buffer
	tags := NewTraceTagsWithWriter(&out)
	tags.Enable("memory")
	if out.Len() != 0 {
		t.Fatal("newly inserted memory tag traced its own table allocation")
	}
	tags.Finalize()
	tags.EnableAll(true)
	tags.Disable("other")
	want := "alloc " + fmt.Sprintf("%p", &tags.tags) +
		" src/util/trace.cpp:31 str_hashtable " + strconv.FormatUint(uint64(reflect.TypeOf(map[string]struct{}{}).Size()), 10) + "\n"
	if !strings.Contains(out.String(), want) || strings.Count(out.String(), "alloc ") != 1 {
		t.Fatal("source allocation trace", out.String(), want)
	}
	tags.Enable("other")
	if strings.Count(out.String(), "alloc ") != 1 {
		t.Fatal("existing table was allocated again")
	}
}
