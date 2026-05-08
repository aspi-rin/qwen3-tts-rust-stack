package audio

import (
	"encoding/binary"
	"math"
)

const (
	SampleRateHz = 24000
	Channels     = 1
)

// F32LEToS16LE converts little-endian float32 PCM samples in [-1, 1] to
// little-endian signed 16-bit PCM. Values outside [-1, 1] are clipped.
func F32LEToS16LE(in []byte) []byte {
	frames := len(in) / 4
	out := make([]byte, frames*2)
	for i := 0; i < frames; i++ {
		bits := binary.LittleEndian.Uint32(in[i*4 : i*4+4])
		f := math.Float32frombits(bits)
		if f > 1 {
			f = 1
		} else if f < -1 {
			f = -1
		}

		var s int16
		if f < 0 {
			s = int16(f * 32768.0)
		} else {
			s = int16(f * 32767.0)
		}
		binary.LittleEndian.PutUint16(out[i*2:i*2+2], uint16(s))
	}
	return out
}
