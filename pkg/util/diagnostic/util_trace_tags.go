// Source: src/util/trace.cpp:25-55; src/util/trace.h:40-46.
package diagnostic

import "strings"

// TraceTags is the caller-owned, single-goroutine trace-tag control state.
// The retained shell and API callers can bind one shared instance.
type TraceTags struct {
	all  bool
	tags map[string]struct{}
}

func NewTraceTags() *TraceTags { return &TraceTags{} }

func traceTagCString(tag string) string {
	if i := strings.IndexByte(tag, 0); i >= 0 {
		return tag[:i]
	}
	return tag
}

func (t *TraceTags) Enable(tag string) {
	if t.tags == nil {
		t.tags = make(map[string]struct{})
	}
	t.tags[traceTagCString(tag)] = struct{}{}
}

func (t *TraceTags) Disable(tag string) {
	if t.tags == nil {
		t.tags = make(map[string]struct{})
	}
	delete(t.tags, traceTagCString(tag))
}

func (t *TraceTags) EnableAll(flag bool) { t.all = flag }

func (t *TraceTags) IsEnabled(tag string) bool {
	if t.all {
		return true
	}
	_, found := t.tags[traceTagCString(tag)]
	return found
}

// Finalize releases the tag table. Source finalize_trace leaves the all flag.
func (t *TraceTags) Finalize() { t.tags = nil }
