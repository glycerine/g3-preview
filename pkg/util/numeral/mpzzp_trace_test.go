package numeral

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestIntegerModularSetupAndInverseSourceTrace(t *testing.T) {
	z, _ := integerManager(t)
	m, err := NewIntegerModularManager(z)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if e := m.Close(); e != nil {
			t.Error(e)
		}
	})
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := m.SetZPUint64(7); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatal("disabled modular trace emitted", output.String())
	}
	tags.Enable("mpzzp")
	tags.Enable("mpzp_inv_bug")
	if err := m.SetZPUint64(8); err != nil {
		t.Fatal(err)
	}
	if err := m.SetZPUint64(7); err != nil {
		t.Fatal(err)
	}
	a := NewInt(3)
	if err := m.Inv(&a); err != nil {
		t.Fatal(err)
	}
	if got, err := z.Int64(&a); err != nil || got != -2 {
		t.Fatal("inverse of 3 modulo 7", got, err)
	}
	if tags.ShouldTrace("mpzzp") {
		const want = "-------- [mpzzp] setup_p src/util/mpzzp.h:55 ---------\n" +
			"lower: -3, upper: 4\n" +
			"------------------------------------------------\n" +
			"-------- [mpzzp] setup_p src/util/mpzzp.h:55 ---------\n" +
			"lower: -3, upper: 3\n" +
			"------------------------------------------------\n" +
			"-------- [mpzp_inv_bug] inv src/util/mpzzp.h:175 ---------\n" +
			"a: 3, p: 7\n" +
			"------------------------------------------------\n" +
			"-------- [mpzp_inv_bug] inv src/util/mpzzp.h:177 ---------\n" +
			"after normalization a: 3\n" +
			"------------------------------------------------\n" +
			"-------- [mpzp_inv_bug] inv src/util/mpzzp.h:179 ---------\n" +
			"tmp1: -2\ntmp2: 1\ntmp3: 1\n" +
			"------------------------------------------------\n"
		if got := output.String(); got != want {
			t.Fatalf("source modular TRACE: got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased modular TRACE emitted", output.String())
	}
}
