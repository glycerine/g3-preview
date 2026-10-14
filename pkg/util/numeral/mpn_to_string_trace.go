// Source: src/util/mpn.cpp:362,395.
package numeral

import (
	"io"
	"strconv"
	"strings"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (m *NaturalManager) traceToCStringInput(a limbView) {
	_ = m.sourceTraceTags().EmitConditional("mpn_to_string", "to_string", "src/util/mpn.cpp", 362,
		func() bool { return true }, func(w io.Writer) error {
			_ = writeNumeralDisplayPart(w, "[mpn] to_string ")
			_ = writeNumeralDisplayPart(w, "[")
			for i := a.size; i > 0; {
				i--
				v, err := readDecimalLimb(a, i)
				if err != nil {
					return err
				}
				_ = writeNumeralDisplayPart(w, strconv.FormatUint(uint64(v), 10))
				if i != 0 {
					_ = writeNumeralDisplayPart(w, "|")
				}
			}
			_ = writeNumeralDisplayPart(w, "]")
			_ = writeNumeralDisplayPart(w, " == ")
			return nil
		})
}

func (m *NaturalManager) traceToCStringResult(dst sourceDecimalDestination) {
	_ = m.sourceTraceTags().EmitConditional("mpn_to_string", "to_string", "src/util/mpn.cpp", 395,
		func() bool { return true }, func(w io.Writer) error {
			var out strings.Builder
			for i := uint64(0); ; i++ {
				c, err := dst.read(i)
				if err != nil {
					return err
				}
				if c == 0 {
					_ = writeNumeralDisplayPart(w, out.String())
					_ = diagnostic.WriteTraceEndl(w)
					return nil
				}
				out.WriteByte(c)
			}
		})
}
