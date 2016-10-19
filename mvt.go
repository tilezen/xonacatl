//go:generate protoc --go_out=. mapnik_vector/vector_tile.proto

package xonacatl

import (
	"io"
	"io/ioutil"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/tilezen/xonacatl/mapnik_vector"
)

func CopyMVTLayers(rd io.Reader, layers map[string]bool, wr io.Writer) error {
	buf, err := ioutil.ReadAll(rd)
	if err != nil {
		return err
	}

	t := &mapnik_vector.Tile{}
	err = proto.Unmarshal(buf, t)
	if err != nil {
		return err
	}

	var new_layers []*mapnik_vector.TileLayer
	for _, l := range t.GetLayers() {
		if *l.Version > 2 {
			return fmt.Errorf("Unable to read layer with version %#q, xonacatl supports versions up to 2 only.", *l.Version)
		}
		if l.Name != nil && layers[*l.Name] {
			new_layers = append(new_layers, l)
		}
	}

	t.Layers = new_layers
	data, err := proto.Marshal(t)
	if err != nil {
		return err
	}

	_, err = wr.Write(data)
	return err
}
