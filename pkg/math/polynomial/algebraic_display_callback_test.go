package polynomial

import (
	"bytes"
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strings"
	"testing"
)

type algebraicDisplayCallbackSink struct {
	algebraicRootTextSink
	callback func()
}

func (s *algebraicDisplayCallbackSink) AppendCString(v string) error {
	if s.callback != nil {
		callback := s.callback
		s.callback = nil
		callback()
	}
	return s.algebraicRootTextSink.AppendCString(v)
}

func TestAlgebraicRootDisplaysRejectInputMutationFromStream(t *testing.T) {
	for _, format := range []string{"root", "smt", "mathematica"} {
		t.Run(format, func(t *testing.T) {
			c, _, _ := algebraicCompareFixture(t)
			root := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
			peer := algebraicCompareRoot(t, c, []int32{-3, 0, 1}, 1, 0, 2, 0, true)
			defer func() { serviceMust(t, c.Del(&root)); serviceMust(t, c.Del(&peer)) }()
			original, originalPeer := root, peer
			calls := 0
			sink := &algebraicDisplayCallbackSink{callback: func() {
				calls++
				for _, call := range []func() error{
					func() error { return c.Del(&root) },
					func() error { return c.CleanupValue(&root) },
					func() error { return c.Swap(&root, &peer) },
					func() error { return c.Copy(&root, &peer) },
				} {
					if e := call(); !errors.Is(e, ErrAlgebraicPending) {
						t.Fatal("stream callback mutated borrowed input", e)
					}
				}
			}}
			var e error
			switch format {
			case "root":
				e = c.DisplayRootInto(sink, &root)
			case "smt":
				e = c.DisplayRootSMT2Into(sink, &root)
			default:
				e = c.DisplayMathematicaInto(sink, &root)
			}
			serviceMust(t, e)
			if calls != 1 || root != original || peer != originalPeer {
				t.Fatal("callback changed actual input identities", calls)
			}
			want := map[string]string{"root": "(#^2 - 2, 2)", "smt": "(root-obj (+ (^ x 2) (- 2)) 2)", "mathematica": "Root[#1^2 - 2 &, 2]"}
			if sink.String() != want[format] {
				t.Fatal("display did not complete original value", sink.String())
			}
		})
	}
}

type algebraicDisplayCallbackWriter struct {
	bytes.Buffer
	callback func()
}

func (w *algebraicDisplayCallbackWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}
func (w *algebraicDisplayCallbackWriter) Write(p []byte) (int, error) {
	if w.callback != nil {
		callback := w.callback
		w.callback = nil
		callback()
	}
	return w.Buffer.Write(p)
}
func TestAlgebraicIntervalDecimalRejectWriterMutation(t *testing.T) {
	for _, format := range []string{"interval", "decimal", "interval-basic", "decimal-basic"} {
		t.Run(format, func(t *testing.T) {
			c, _, _ := algebraicCompareFixture(t)
			root := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
			basic := strings.HasSuffix(format, "-basic")
			if basic {
				q := unumeral.NewRational()
				serviceMust(t, c.config.Rationals.SetFraction(&q, 3, 2))
				serviceMust(t, c.SetRational(&root, &q))
				serviceMust(t, c.config.Rationals.Del(&q))
			}
			peer := algebraicCompareRoot(t, c, []int32{-3, 0, 1}, 1, 0, 2, 0, true)
			defer func() { serviceMust(t, c.Del(&root)); serviceMust(t, c.Del(&peer)) }()
			original, originalPeer := root, peer
			calls := 0
			writer := &algebraicDisplayCallbackWriter{callback: func() {
				calls++
				for _, call := range []func() error{
					func() error { return c.Del(&root) },
					func() error { return c.CleanupValue(&root) },
					func() error { return c.Swap(&root, &peer) },
					func() error { return c.Copy(&root, &peer) },
					c.Close, c.CleanupRetained,
				} {
					if e := call(); !errors.Is(e, ErrAlgebraicPending) {
						t.Fatal("writer callback obtained executing owner", e)
					}
				}
			}}
			var e error
			want := "(1, 2)"
			if strings.HasPrefix(format, "interval") {
				e = c.DisplayInterval(writer, &root)
				if basic {
					want = "[3/2, 3/2]"
				}
			} else {
				e = c.DisplayDecimal(writer, &root, 4)
				want = "1.4142?"
				if basic {
					want = "1.5"
				}
			}
			serviceMust(t, e)
			if calls != 1 || root != original || peer != originalPeer || writer.String() != want {
				t.Fatal("writer changed input or output", calls, writer.String(), want)
			}
		})
	}
}
