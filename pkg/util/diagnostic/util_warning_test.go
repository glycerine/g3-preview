package diagnostic

import (
	"bytes"
	"strings"
	"testing"
)

func TestWarningSourceDefaultRoutingAndDisabledState(t *testing.T) {
	var stdout, stderr bytes.Buffer
	s := NewWarningService(&stdout, &stderr)
	if !s.Enabled() {
		t.Fatal("source warning default")
	}
	if e := s.Warning("first\x00ignored"); e != nil {
		t.Fatal(e)
	}
	if got := stderr.String(); got != "WARNING: first\n" {
		t.Fatal(got)
	}
	if e := s.SendToStdout(true); e != nil {
		t.Fatal(e)
	}
	if e := s.Warning("second"); e != nil {
		t.Fatal(e)
	}
	if got := stdout.String(); got != "WARNING: second\n" {
		t.Fatal(got)
	}
	if e := s.Enable(false); e != nil {
		t.Fatal(e)
	}
	if e := s.Warning("suppressed"); e != nil {
		t.Fatal(e)
	}
	if got := stdout.String(); got != "WARNING: second\n" {
		t.Fatal(got)
	}
}

func TestWarningSourceBorrowedStreamTruncationAndDisplayer(t *testing.T) {
	var stdout, stderr, diagnostic bytes.Buffer
	s := NewWarningService(&stdout, &stderr)
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
	d, e := NewWarningDisplayer(s, "once")
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Enable(false); e != nil {
		t.Fatal(e)
	}
	if e = d.Sign(); e != nil {
		t.Fatal(e)
	}
	if e = s.Enable(true); e != nil {
		t.Fatal(e)
	}
	if e = d.Sign(); e != nil {
		t.Fatal(e)
	}
	if diagnostic.String() != "WARNING: "+long[:1023]+"\n" {
		t.Fatal("disabled sign must still consume one-shot display")
	}
	d.Reset()
	if e = d.Sign(); e != nil {
		t.Fatal(e)
	}
	if got := diagnostic.String(); got != "WARNING: "+long[:1023]+"\nWARNING: once\n" {
		t.Fatal(got)
	}
	if e = s.SetWarningStream(nil); e != nil {
		t.Fatal(e)
	}
	if e = s.Warning(long); e != nil {
		t.Fatal(e)
	}
	if got := stderr.String(); got != "WARNING: "+long+"\n" {
		t.Fatal("default FILE route must not truncate", len(got))
	}
}
