package xonacatl

import "io"

type LayerCopier interface {
	CopyLayers(io.Reader, io.Writer) error
}
