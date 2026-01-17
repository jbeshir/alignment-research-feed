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

func TestAddVectors(t *testing.T) {
	cases := []struct {
		name     string
		a        []float32
		b        []float32
		expected []float32
		wantErr  bool
	}{
		{
			name:     "simple_add",
			a:        []float32{1.0, 2.0, 3.0},
			b:        []float32{0.5, 0.5, 0.5},
			expected: []float32{1.5, 2.5, 3.5},
		},
		{
			name:     "with_negatives",
			a:        []float32{1.0, -2.0, 3.0},
			b:        []float32{-1.0, 2.0, -3.0},
			expected: []float32{0.0, 0.0, 0.0},
		},
		{
			name:     "empty_vectors",
			a:        []float32{},
			b:        []float32{},
			expected: []float32{},
		},
		{
			name:    "length_mismatch",
			a:       []float32{1.0, 2.0},
			b:       []float32{1.0, 2.0, 3.0},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := addVectors(tc.a, tc.b)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "vector length mismatch")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestSubtractVectors(t *testing.T) {
	cases := []struct {
		name     string
		a        []float32
		b        []float32
		expected []float32
		wantErr  bool
	}{
		{
			name:     "simple_subtract",
			a:        []float32{1.5, 2.5, 3.5},
			b:        []float32{0.5, 0.5, 0.5},
			expected: []float32{1.0, 2.0, 3.0},
		},
		{
			name:     "result_negative",
			a:        []float32{1.0, 2.0, 3.0},
			b:        []float32{2.0, 3.0, 4.0},
			expected: []float32{-1.0, -1.0, -1.0},
		},
		{
			name:     "empty_vectors",
			a:        []float32{},
			b:        []float32{},
			expected: []float32{},
		},
		{
			name:    "length_mismatch",
			a:       []float32{1.0, 2.0, 3.0},
			b:       []float32{1.0, 2.0},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := subtractVectors(tc.a, tc.b)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "vector length mismatch")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}
