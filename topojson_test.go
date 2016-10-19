package xonacatl

import (
	"testing"
	"strings"
	"bytes"
)

func runCopyTopoJSON(input string, layers map[string]bool) (output string, err error) {
	var buf bytes.Buffer
	rd := strings.NewReader(input)

	err = CopyTopoJSONLayers(rd, layers, &buf)
	if err == nil {
		output = buf.String()
	}

	return
}

func runCopyTopoJSONSuccess(input string, layers map[string]bool, t *testing.T) string {
	out, err := runCopyTopoJSON(input, layers)
	if err != nil {
		t.Fatalf("CopyTopoJSONLayers(%#v) failed, error: %s", input, err.Error())
	}
	return out
}

func runCopyTopoJSONAssertOutput(input string, layers map[string]bool, expected string, t *testing.T) {
	out := runCopyTopoJSONSuccess(input, layers, t)
	// get rid of spurious failures due to trailing whitespace
	out = strings.TrimSpace(out)
	if out != expected {
		t.Fatalf("Expected output of CopyTopoJSONLayers(%#v) to be %#v, but instead was %#v", input, expected, out)
	}
}

const (
	minimal = `{"type":"Topology","objects":{},"arcs":[]}`
	foo = `{"type":"Topology","objects":{"foo":{"foo":false}},"arcs":[]}`
	foobar = `{"type":"Topology","objects":{"bar":{"bar":false},"foo":{"foo":false}},"arcs":[]}`
)

func TestTopoJSONEmptyWithoutLayers(t *testing.T) {
	runCopyTopoJSONAssertOutput(minimal, map[string]bool{}, minimal, t)
}

func TestTopoJSONEmptyWithLayers(t *testing.T) {
	runCopyTopoJSONAssertOutput(minimal, map[string]bool{"foo":true}, minimal, t)
}

func TestTopoJSONNonEmptyWithoutLayers(t *testing.T) {
	runCopyTopoJSONAssertOutput(foobar, map[string]bool{}, minimal, t)
}

func TestTopoJSONNonEmptyWithSingleLayer(t *testing.T) {
	runCopyTopoJSONAssertOutput(foobar, map[string]bool{"foo":true}, foo, t)
}

func TestTopoJSONNonEmptyWithLayers(t *testing.T) {
	runCopyTopoJSONAssertOutput(foobar, map[string]bool{"foo":true,"bar":true}, foobar, t)
}
