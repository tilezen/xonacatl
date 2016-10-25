package main

import (
	"testing"
	"regexp"
)

func doNotForward(t *testing.T, h *LayersHandler, header string) {
	if h.forwardHeader(header) {
		t.Fatalf("Should not forward header %#v, but h.forwardHeader returned true.", header)
	}
}

func doForward(t *testing.T, h *LayersHandler, header string) {
	if !h.forwardHeader(header) {
		t.Fatalf("Should forward header %#v, but h.forwardHeader returned false.", header)
	}
}

func TestForwardHeader(t *testing.T) {
	h := &LayersHandler{
		do_not_forward_headers: []*regexp.Regexp{regexp.MustCompile("(?i)X-Mz-*")},
	}

	doNotForward(t, h, "x-mz-foo")
	doNotForward(t, h, "X-MZ-FOO")

	doForward(t, h, "xmz-foo")
}
