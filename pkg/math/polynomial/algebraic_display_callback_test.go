package polynomial

import (
	"errors"
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
