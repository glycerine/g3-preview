package diagnostic

import (
	"bytes"
	"strings"
	"testing"
)

func TestWarningSourceDefaultRoutingAndDisabledState(t *testing.T) {
	var stderr bytes.Buffer
	s := NewWarningService(&stderr)
	if !s.Enabled() {
		t.Fatal("source warning default")
	}
	if e := s.Warning("first\x00ignored"); e != nil {
		t.Fatal(e)
	}
	if got := stderr.String(); got != "WARNING: first\n" {
		t.Fatal(got)
	}
	if e := s.Enable(false); e != nil {
		t.Fatal(e)
	}
	if e := s.Warning("suppressed"); e != nil {
		t.Fatal(e)
	}
	if got := stderr.String(); got != "WARNING: first\n" {
		t.Fatal(got)
	}
}

func TestWarningSourceBorrowedStreamTruncationAndFallback(t *testing.T) {
	var stderr, diagnostic bytes.Buffer
	s := NewWarningService(&stderr)
	if e := s.SetWarningStream(&diagnostic); e != nil {
		t.Fatal(e)
	}
	long := strings.Repeat("x", 1100)
	if e := s.Warning(long); e != nil {
		t.Fatal(e)
	}
	if got := diagnostic.String(); got != "WARNING: "+long[:1023]+"\n" {
		t.Fatal("source 1024-byte ostream buffer", len(got))
	}
	if e := s.SetWarningStream(nil); e != nil {
		t.Fatal(e)
	}
	if e := s.Warning(long); e != nil {
		t.Fatal(e)
	}
	if got := stderr.String(); got != "WARNING: "+long+"\n" {
		t.Fatal("default FILE route must not truncate", len(got))
	}
}
