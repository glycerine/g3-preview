package smtcore

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func TestSMTStringValueEncodingCompleteByteAndTwoByteSpaces(t *testing.T) {
	_, theory, _, _ := stringLibraryFixture(t)
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 !\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
	wantAlphabet := []byte(alphabet)
	for b := 1; b <= 31; b++ {
		wantAlphabet = append(wantAlphabet, byte(b))
	}
	for b := 127; b <= 255; b++ {
		wantAlphabet = append(wantAlphabet, byte(b))
	}
	if len(wantAlphabet) != 255 {
		t.Fatal("invalid independent alphabet", len(wantAlphabet))
	}
	for length := 0; length <= 2; length++ {
		base := make([]int32, length+1)
		count := 1
		for i := 0; i < length; i++ {
			count *= 255
		}
		for ordinal := 0; ordinal < count; ordinal++ {
			word, err := theory.stringValueFromEncoding(int32(length), base)
			astTestMust(t, err)
			want := make([]byte, length)
			index := ordinal
			for i := length - 1; i >= 0; i-- {
				want[i] = wantAlphabet[index%255]
				index /= 255
			}
			if word != string(want) || strings.IndexByte(word, 0) >= 0 {
				t.Fatalf("length %d ordinal %d: %q want %q", length, ordinal, word, want)
			}
			saved := append([]int32(nil), base...)
			next, covered, err := theory.nextStringValueEncoding(base)
			astTestMust(t, err)
			if !reflect.DeepEqual(base, saved) || covered != (ordinal == count-1) {
				t.Fatal("carry mutated input or ended early", length, ordinal)
			}
			if covered {
				if next != nil {
					t.Fatal("exhausted encoding must reset output")
				}
			} else {
				base[0] = -1
				if next[0] < 0 {
					t.Fatal("next encoding aliases input")
				}
				base = next
			}
		}
	}
}

func TestSMTStringValueEncodingRejectsInvalidProducerState(t *testing.T) {
	_, theory, _, _ := stringLibraryFixture(t)
	for _, test := range []struct {
		length   int32
		encoding []int32
	}{
		{-1, nil}, {0, nil}, {1, []int32{0}}, {0, []int32{1}}, {1, []int32{-1, 0}}, {1, []int32{255, 0}},
	} {
		if _, err := theory.stringValueFromEncoding(test.length, test.encoding); !errors.Is(err, ast.ErrASTDomain) {
			t.Fatal("invalid word encoding accepted", test, err)
		}
	}
	for _, base := range [][]int32{nil, {1}, {-1, 0}, {255, 0}, {0, 1}} {
		if _, _, err := theory.nextStringValueEncoding(base); !errors.Is(err, ast.ErrASTDomain) {
			t.Fatal("invalid carry encoding accepted", base, err)
		}
	}
}
