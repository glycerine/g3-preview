package memory

import "testing"

func TestRuntimeSourceMegabytesToBytesSaturatesAtSizeT(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		m, _ := memoryRuntimeFixture(t, false, bits)
		for _, x := range []struct {
			mb   uint32
			want uint64
		}{
			{0, 0},
			{1, 1 << 20},
			{4095, 4095 << 20},
			{4096, map[uint8]uint64{32: 1<<32 - 1, 64: 1 << 32}[bits]},
			{^uint32(0), map[uint8]uint64{32: 1<<32 - 1, 64: ^uint64(0)}[bits]},
		} {
			memoryRuntimeRequire(t, m.SetMaxSizeMegabytes(x.mb))
			if got := uint64(m.accounting.maxSize); got != x.want {
				t.Fatalf("size_t%d megabytes %d: %d want %d", bits, x.mb, got, x.want)
			}
		}
	}
}
