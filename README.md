Xonacatl
========

Xonacatl is a "layers server", which takes requests for a [subset of layers](https://mapzen.com/projects/vector-tiles/#layers) available and requests all the layers upstream, stripping out the unwanted layers when writing the response back to the client.

For example:

1. The client requests a tile `/buildings,water/0/0/0.json`
2. Xonacatl makes an upstream request for `/all/0/0/0.json`
3. Xonacatl ignores anything in the GeoJSON response which isn't for `buildings` or `water` layers.
4. The client reads back a tile containing only the layers they asked for.

Why?
----

Custom layers can be very helpful in reducing tile download size. Some layers, particularly `buildings` and `landuse`, can be large at certain zooms. Downloading this data unnecessarily can make apps feel less responsive.

On the other hand, creating tiles with custom layers is time-consuming and resource-intensive. It's much faster and cheaper to "cut out" the layers which the client wants from a tile containing all the layers. In fact, this is how [tileserver](https://github.com/tilezen/tileserver) works. Unfortunately, custom layers are customised, and the greater variety of their URLs means that cache hit ratios are reduced - on top of the additional latency of returning the request to the origin server.

Xonacatl is an attempt to do this at the "edge", closer to the client and taking advantage of as much "edge" caching as possible.

Why "xonacatl"?
---------------

[Xonacatl](https://en.wiktionary.org/wiki/xonacatl) is Nahuatl for onion, and [tiles are like onions](http://www.imdb.com/title/tt0126029/quotes?item=qt0398107).

Installing
----------

```
cd $GOPATH
go get -u github.com/golang/protobuf/proto
go get -u github.com/tilezen/xonacatl/xonacatl_server
go install github.com/tilezen/xonacatl/xonacatl_server
```

To update the generated protocol buffers code, you will need to run `go generate` and have the protobuf Go compiler plugin installed:

```
cd $GOPATH
go get -u github.com/golang/protobuf/protoc-gen-go
PATH=$PATH:$PWD/bin go generate github.com/tilezen/xonacatl
```

As is the Go convention, the generated source code is checked in. Unless you need to alter the original protocol buffers definiton, it shoud not be necessary to install the `protoc-gen-go` tool, or run `go generate`.
