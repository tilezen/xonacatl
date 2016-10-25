package xonacatl

import (
	"encoding/json"
	"io"
)

type topoObject struct {
	data json.RawMessage
}

func (t *topoObject) MarshalJSON() ([]byte, error) {
	return t.data.MarshalJSON()
}

func (t *topoObject) UnmarshalJSON(data []byte) error {
	return t.data.UnmarshalJSON(data)
}

type topoJSON struct {
	Type      string                 `json:"type"`
	Transform *json.RawMessage       `json:"transform,omitempty"`
	Objects   map[string]*topoObject `json:"objects"`
	// arcs currently stored as RawMessage to avoid precision issues when Unmarshalling and Marshalling them. see long comment in json.go.
	Arcs      *json.RawMessage       `json:"arcs"`
}

type topoJSONCopier struct {
	layers map[string]bool
}

func NewCopyTopoJSONLayers(layers map[string]bool) *topoJSONCopier {
	return &topoJSONCopier{layers: layers}
}

func (c *topoJSONCopier) CopyLayers(rd io.Reader, wr io.Writer) error {
	var t topoJSON

	dec := json.NewDecoder(rd)

	err := dec.Decode(&t)
	if err != nil {
		return err
	}

	for k := range t.Objects {
		if !c.layers[k] {
			delete(t.Objects, k)
		}
	}

	// TODO: collect arcs and reset unused ones to empty

	enc := json.NewEncoder(wr)
	enc.SetIndent("", "")
	return enc.Encode(&t)
}
