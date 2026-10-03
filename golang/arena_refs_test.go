package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

// Storage adaptation tests, added to implement GO_PORT_PLAN B6. No AST manager
// or solver behavior is claimed by these handle representation checks.
func TestInternalRefRoundTrip(t *testing.T) {
	for _, slot := range []uint32{1, 255, 1 << 24, 0xffffffff} {
		for _, generation := range []uint32{1, 19, 0xffffffff} {
			r := makeRef(arena.Slot(slot), arena.Generation(generation))
			if refSlot(r) != arena.Slot(slot) {
				t.Fatalf("slot truncated: %x", r)
			}
			if checkedRefs && refGeneration(r) != arena.Generation(generation) {
				t.Fatalf("generation truncated: %x", r)
			}
			if !checkedRefs && refGeneration(r) != 0 {
				t.Fatal("release claims a generation")
			}
		}
	}
	if refSlot(0) != 0 || refGeneration(0) != 0 {
		t.Fatal("null decoding")
	}
}
func TestInternalAndPublicReuse(t *testing.T) {
	s, err := arena.NewSlab[uint32](arena.Config{PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	slot, g, err := s.Allocate()
	if err != nil {
		t.Fatal(err)
	}
	r := makeRef(slot, g)
	token := makeToken(slot, g)
	if !validRef(s, r) || !validToken(s, token) {
		t.Fatal("fresh handle invalid")
	}
	if err := s.Recycle(slot); err != nil {
		t.Fatal(err)
	}
	if validRef(s, r) || validToken(s, token) {
		t.Fatal("dead handle valid")
	}
	slot2, g2, err := s.Allocate()
	if err != nil {
		t.Fatal(err)
	}
	if slot2 != slot || g2 == g {
		t.Fatal("no reuse or generation advance")
	}
	if validToken(s, token) {
		t.Fatal("public stale token accepted")
	}
	if checkedRefs && validRef(s, r) {
		t.Fatal("checked stale reference accepted")
	}
	if !checkedRefs && !validRef(s, r) {
		t.Fatal("release slot representation changed")
	}
	if !validRef(s, makeRef(slot2, g2)) || !validToken(s, makeToken(slot2, g2)) {
		t.Fatal("new handles invalid")
	}
	s.Close()
	if validToken(s, makeToken(slot2, g2)) {
		t.Fatal("closed token accepted")
	}
}
