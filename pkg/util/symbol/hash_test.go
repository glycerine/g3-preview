// Source: src/util/hash.cpp; src/util/symbol.cpp:48-52.
package symbol

import (
	_ "embed"
	"encoding/json"
	"testing"
)

// Existing C++ golden values, little-endian words and signed char tails.
//
//go:embed test_vectors/string_hash.json
var spellingHashVectors []byte

func TestSymbolSpellingHashSourceGolden(t *testing.T) {
	var want []uint32
	if e := json.Unmarshal(spellingHashVectors, &want); e != nil {
		t.Fatal(e)
	}
	var bytes [25]byte
	for i := range bytes {
		bytes[i] = byte(i*19 + 127)
	}
	for n, w := range want {
		if got := spellingHash(string(bytes[:n]), 123); got != w {
			t.Fatalf("length %d: got %d want %d", n, got, w)
		}
	}
}
