package xonacatl

import (
	"bytes"
	"strings"
	"testing"
)

func runCopy(input string, layers map[string]bool) (output string, err error) {
	var buf bytes.Buffer
	rd := strings.NewReader(input)

	copier := NewCopyLayers(layers)
	err = copier.CopyLayers(rd, &buf)
	if err == nil {
		output = buf.String()
	}

	return
}

func runCopySuccess(input string, layers map[string]bool, t *testing.T) string {
	out, err := runCopy(input, layers)
	if err != nil {
		t.Fatalf("CopyLayers(%#v) failed, error: %s", input, err.Error())
	}
	return out
}

func runCopyAssertOutput(input string, layers map[string]bool, expected string, t *testing.T) {
	out := runCopySuccess(input, layers, t)
	if out != expected {
		t.Fatalf("Expected output of CopyLayers(%#v) to be %#v, but instead was %#v", input, expected, out)
	}
}

func TestEmptyWithoutLayers(t *testing.T) {
	runCopyAssertOutput("{}", map[string]bool{}, "{}", t)
}

func TestEmptyWithLayers(t *testing.T) {
	runCopyAssertOutput("{}", map[string]bool{"foo": true}, "{}", t)
}

func TestNonEmptyWithoutLayers(t *testing.T) {
	runCopyAssertOutput("{\"foo\":{\"bar\":false}}", map[string]bool{}, "{}", t)
}

func TestNonEmptyWithSingleLayer(t *testing.T) {
	json := "{\"foo\":{\"bar\":false},\"zzz\":false}"
	runCopyAssertOutput(json, map[string]bool{"foo": true}, "{\"bar\":false}", t)
}

func TestNonEmptyWithLayers(t *testing.T) {
	json := "{\"foo\":{\"bar\":false},\"zzz\":false}"
	runCopyAssertOutput(json, map[string]bool{"foo": true, "zzz": true}, json, t)
}
