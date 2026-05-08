package audio

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestF32LEToS16LE(t *testing.T) {
	floats := []float32{-2, -1, -0.5, 0, 0.5, 1, 2}
	in := make([]byte, len(floats)*4)
	for i, f := range floats {
		binary.LittleEndian.PutUint32(in[i*4:i*4+4], math.Float32bits(f))
	}

	out := F32LEToS16LE(in)
	got := make([]int16, len(floats))
	for i := range got {
		got[i] = int16(binary.LittleEndian.Uint16(out[i*2 : i*2+2]))
	}
	want := []int16{-32768, -32768, -16384, 0, 16383, 32767, 32767}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sample %d: got %d, want %d", i, got[i], want[i])
		}
	}
}
