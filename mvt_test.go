package xonacatl

import (
	"bytes"
	"testing"
)

func runCopyMVT(input []byte, layers map[string]bool) (output []byte, err error) {
	var buf bytes.Buffer
	rd := bytes.NewBuffer(input)

	copier := NewCopyMVTLayers(layers)
	err = copier.CopyLayers(rd, &buf)
	if err == nil {
		output = buf.Bytes()
	}

	return
}

func runCopyMVTSuccess(input []byte, layers map[string]bool, t *testing.T) []byte {
	out, err := runCopyMVT(input, layers)
	if err != nil {
		t.Fatalf("CopyMVTLayers(%#v) failed, error: %s", input, err.Error())
	}
	return out
}

func byteSliceEq(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func runCopyMVTAssertOutput(input []byte, layers map[string]bool, expected []byte, t *testing.T) {
	out := runCopyMVTSuccess(input, layers, t)
	if !byteSliceEq(out, expected) {
		t.Fatalf("Expected output of CopyMVTLayers(%#v) to be %#v, but instead was %#v", input, expected, out)
	}
}

func TestMVTEmptyWithoutLayers(t *testing.T) {
	runCopyMVTAssertOutput([]byte{}, map[string]bool{}, []byte{}, t)
}

func TestMVTEmptyWithLayers(t *testing.T) {
	runCopyMVTAssertOutput([]byte{}, map[string]bool{"foo": true}, []byte{}, t)
}

func TestMVTNonEmptyWithoutLayers(t *testing.T) {
	// has a water layer with a single feature.
	mvt := []byte{26, 73, 10, 5, 119, 97, 116, 101, 114, 18, 26, 8, 1, 18, 6, 0, 0, 1, 1, 2, 2, 24, 3, 34, 12, 9, 0, 128, 64, 26, 0, 1, 2, 0, 0, 2, 15, 26, 3, 102, 111, 111, 26, 3, 98, 97, 122, 26, 3, 117, 105, 100, 34, 5, 10, 3, 98, 97, 114, 34, 5, 10, 3, 102, 111, 111, 34, 2, 32, 123, 40, 128, 32, 120, 2}
	runCopyMVTAssertOutput(mvt, map[string]bool{}, []byte{}, t)
}

func TestMVTNonEmptyWithLayers(t *testing.T) {
	// has a water layer with a single feature.
	mvt := []byte{26, 73, 10, 5, 119, 97, 116, 101, 114, 18, 26, 8, 1, 18, 6, 0, 0, 1, 1, 2, 2, 24, 3, 34, 12, 9, 0, 128, 64, 26, 0, 1, 2, 0, 0, 2, 15, 26, 3, 102, 111, 111, 26, 3, 98, 97, 122, 26, 3, 117, 105, 100, 34, 5, 10, 3, 98, 97, 114, 34, 5, 10, 3, 102, 111, 111, 34, 2, 32, 123, 40, 128, 32, 120, 2}
	runCopyMVTAssertOutput(mvt, map[string]bool{"water": true}, mvt, t)
}
