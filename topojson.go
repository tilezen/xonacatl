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
	Arcs      *json.RawMessage       `json:"arcs"`
}

func CopyTopoJSONLayers(rd io.Reader, layers map[string]bool, wr io.Writer) error {
	var t topoJSON

	dec := json.NewDecoder(rd)

	err := dec.Decode(&t)
	if err != nil {
		return err
	}

	for k := range t.Objects {
		if !layers[k] {
			delete(t.Objects, k)
		}
	}

	// TODO: collect arcs and reset unused ones to empty

	enc := json.NewEncoder(wr)
	enc.SetIndent("", "")
	return enc.Encode(&t)
}
