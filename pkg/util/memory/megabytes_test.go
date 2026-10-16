package memory

import "testing"

func TestSourceMegabytesToBytesNativeAndNarrowSizeT(t *testing.T) {
	maxNative := uint64(^uint(0))
	maxNarrow := uint64(^uint32(0))
	for _, tc := range []struct {
		mb     uint32
		native uint64
		narrow uint64
	}{
		{0, 0, 0},
		{1, 1 << 20, 1 << 20},
		{4095, 4095 << 20, 4095 << 20},
		{4096, 4096 << 20, maxNarrow},
		{^uint32(0) - 1, uint64(^uint32(0)-1) << 20, maxNarrow},
		{^uint32(0), maxNative, maxNarrow},
	} {
		wantHost := tc.native
		if maxNative == maxNarrow {
			wantHost = tc.narrow
		}
		if got := MegabytesToBytes(tc.mb); got != wantHost {
			t.Fatalf("native %d MB: got %d, want %d", tc.mb, got, wantHost)
		}
		if got := megabytesToBytesLimit(tc.mb, maxNarrow); got != tc.narrow {
			t.Fatalf("narrow %d MB: got %d, want %d", tc.mb, got, tc.narrow)
		}
	}
}
