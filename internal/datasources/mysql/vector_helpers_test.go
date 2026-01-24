package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFloat32SliceToBytes_RoundTrip(t *testing.T) {
	cases := []struct {
		name   string
		floats []float32
	}{
		{
			name:   "empty",
			floats: []float32{},
		},
		{
			name:   "single",
			floats: []float32{1.5},
		},
		{
			name:   "multiple",
			floats: []float32{0.1, 0.2, 0.3, -0.5, 100.0},
		},
		{
			name:   "zeros",
			floats: []float32{0.0, 0.0, 0.0},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bytes := float32SliceToBytes(tc.floats)
			result, err := bytesToFloat32Slice(bytes)
			require.NoError(t, err)
			assert.Equal(t, tc.floats, result)
		})
	}
}

func TestBytesToFloat32Slice_InvalidLength(t *testing.T) {
	cases := []struct {
		name  string
		bytes []byte
	}{
		{
			name:  "one_byte",
			bytes: []byte{0x01},
		},
		{
			name:  "three_bytes",
			bytes: []byte{0x01, 0x02, 0x03},
		},
		{
			name:  "five_bytes",
			bytes: []byte{0x01, 0x02, 0x03, 0x04, 0x05},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := bytesToFloat32Slice(tc.bytes)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid byte length")
		})
	}
}
