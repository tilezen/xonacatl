package xonacatl

import (
	"testing"
	"bytes"
)

func runCopyMVT(input []byte, layers map[string]bool) (output []byte, err error) {
	var buf bytes.Buffer
	rd := bytes.NewBuffer(input)

	err = CopyMVTLayers(rd, layers, &buf)
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
	if a == nil && b == nil {
		return true

	} else if a == nil || b == nil {
		return false

	} else if len(a) != len(b) {
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
	if byteSliceEq(out, expected) {
		t.Fatalf("Expected output of CopyMVTLayers(%#v) to be %#v, but instead was %#v", input, expected, out)
	}
}

func TestMVTEmptyWithoutLayers(t *testing.T) {
	runCopyMVTAssertOutput([]byte{}, map[string]bool{}, []byte{}, t)
}

func TestMVTEmptyWithLayers(t *testing.T) {
	runCopyMVTAssertOutput([]byte{}, map[string]bool{"foo":true}, []byte{}, t)
}
