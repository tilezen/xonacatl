package xonacatl

import (
	"encoding/json"
	"fmt"
	"io"
)

func assertDelim(dec *json.Decoder, r json.Delim) error {
	t, err := dec.Token()
	if err != nil {
		return err
	}

	b, ok := t.(json.Delim)
	if !ok {
		return fmt.Errorf("Unexpected token %#v, expected Delim %#v", t, r)
	}

	if b != r {
		return fmt.Errorf("Unexpected delimiter %#v, expecting %#v", b, r)
	}

	return nil
}

type LayersWriter struct {
	wr          io.Writer
	multi_layer bool
	layer       int
	did_write   bool
}

func (w *LayersWriter) WriteLayer(k string, m *json.RawMessage) error {
	var err error

	if w.multi_layer {
		if w.layer > 0 {
			_, err = io.WriteString(w.wr, ",")
			if err != nil {
				return err
			}
		}
		w.layer += 1

		var bytes []byte
		bytes, err = json.Marshal(k)
		if err != nil {
			return err
		}

		_, err = w.wr.Write(bytes)
		if err != nil {
			return err
		}

		_, err = io.WriteString(w.wr, ":")
		if err != nil {
			return err
		}
	}

	_, err = w.wr.Write(*m)
	w.did_write = true
	return err
}

func (w *LayersWriter) Begin() error {
	return w.writeDelim("{")
}

func (w *LayersWriter) End() error {
	if !w.multi_layer && !w.did_write {
		_, err := io.WriteString(w.wr, "{}")
		return err
	}
	return w.writeDelim("}")
}

func (w *LayersWriter) writeDelim(s string) error {
	if w.multi_layer {
		_, err := io.WriteString(w.wr, s)
		if err != nil {
			return err
		}
	}
	return nil
}

type geoJSONCopier struct {
	layers map[string]bool
}

func NewCopyLayers(layers map[string]bool) *geoJSONCopier {
	return &geoJSONCopier{layers: layers}
}

func (c *geoJSONCopier) CopyLayers(rd io.Reader, wr io.Writer) error {
	var err error
	var num_layers int

	// we use a streaming model here to avoid having to Unmarshal the whole document. the upstream server goes to some length to use the right number of digits of precision for the floating point coordinates depending on the zoom level. Unmarshalling and re-Marshalling here would mean treating that very carefully to ensure the same behaviour, whereas a streaming approach means we can ignore it and stream back the bytes of the original document unmodified. also, Unmarshal and Marshal can be quite time and memory consuming because of the size of the JSON tree, so avoiding them is a double benefit.
	dec := json.NewDecoder(rd)

	num_layers = 0
	for _, v := range c.layers {
		if v {
			num_layers += 1
		}
	}

	if num_layers == 0 {
		_, err = io.WriteString(wr, "{}")
		return err
	}
	enc := &LayersWriter{
		wr:          wr,
		multi_layer: num_layers > 1,
		layer:       0,
	}

	assertDelim(dec, '{')
	err = enc.Begin()
	if err != nil {
		return err
	}

	for dec.More() {
		var tok json.Token
		// the RawMessage value means the document is parsed, but doesn't create an in-memory representation of the JSON document as Unmarshal does. this is both faster and avoids issues around precision of floating point numbers (see longer comment above).
		var m json.RawMessage

		tok, err = dec.Token()
		if err != nil {
			return err
		}
		k, ok := tok.(string)
		if !ok {
			return fmt.Errorf("Expecting string object key, found %#v", tok)
		}

		err = dec.Decode(&m)
		if err != nil {
			return err
		}

		if c.layers[k] {
			err = enc.WriteLayer(k, &m)
			if err != nil {
				return err
			}
		}
	}

	assertDelim(dec, '}')
	err = enc.End()
	if err != nil {
		return err
	}

	return nil
}
