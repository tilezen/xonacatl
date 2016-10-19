package main

import (
	"github.com/tilezen/xonacatl"
	"github.com/namsral/flag"
	"github.com/gorilla/mux"
	"net/http"
	"log"
	"net/url"
	"io"
	"os"
	"strings"
)

type LayersHandler struct {
	origin *url.URL
	route *mux.Route
}

type CopyFunc func(io.Reader, map[string]bool, io.Writer) error

func copyAll(rd io.Reader, _ map[string]bool, wr io.Writer) error {
	_, err := io.Copy(wr, rd)
	return err
}

func (h *LayersHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var request_layers string
	var format string

	var pairs []string
	for k, v := range mux.Vars(req) {
		// override the layers, save the old value
		if k == "layers" {
			request_layers = v
			v = "all"

		} else if k == "format" {
			format = v
		}

		pairs = append(pairs, k, v)
	}

	origin_path, err := h.route.URLPath(pairs...)
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}

	origin_url := *h.origin
	origin_url.Path = origin_path.Path
	// copy request paramters, as this might include API key
	values := make(url.Values)
	for k, vs := range req.Form {
		for _, v := range vs {
			values.Add(k, v)
		}
	}
	origin_url.RawQuery = values.Encode()

	new_req, err := http.NewRequest(req.Method, origin_url.String(), req.Body)
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}

	for k, v := range req.Header {
		new_req.Header[k] = v
	}
	// TODO: override User-Agent? Add X-Forwarded-For?
	resp, err := http.DefaultClient.Do(new_req)
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}

	for k, v := range resp.Header {
		rw.Header()[k] = v
	}
	rw.WriteHeader(resp.StatusCode)

	layers := make(map[string]bool)
	for _, l := range strings.Split(request_layers, ",") {
		layers[l] = true
	}

	var copy_func CopyFunc
	if request_layers == "all" {
		copy_func = copyAll

	} else if format == "json" {
		copy_func = xonacatl.CopyLayers

	} else if format == "mvt" || format == "mvtb" {
		copy_func = xonacatl.CopyMVTLayers

	} else {
		// fall back to just copying the request as-is
		copy_func = copyAll
	}

	err = copy_func(resp.Body, layers, rw)

	// possibly can't return this to the client, as we've already written the
	// response header. a write failure at this stage also could be an error
	// writing _to_ the client.
	if err != nil {
		log.Printf("Error while writing response body: %s", err.Error())
	}
}

func main() {
	var pattern, origin, listen string

	f := flag.NewFlagSetWithEnvPrefix(os.Args[0], "XONACATL", 0)
	f.StringVar(&pattern, "pattern", "/mapzen/v{version:[0-9]+}/{layers}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}.{fmt}", "pattern to use when matching incoming tile requests")
	f.StringVar(&origin, "host", "http://tile.mapzen.com/mapzen/v{version:[0-9]+}/{layers}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}.{fmt}", "URL pattern to fetch tiles from")
	f.StringVar(&listen, "listen", ":8080", "interface and port to listen on")
	err := f.Parse(os.Args[1:])
	if err == flag.ErrHelp {
		return
	} else if err != nil {
		log.Fatalf("Unable to parse input command line, environment or config: %s", err.Error())
	}

	url, err := url.Parse(origin)
	if err != nil {
		log.Fatalf("Unable to parse origin URL: %s", err.Error())
	}

	origin_router := mux.NewRouter()
	origin_router.NewRoute().Path(url.Path).BuildOnly().Name("origin")

	h := &LayersHandler{
		origin: url,
		route: origin_router.GetRoute("origin"),
	}

	r := mux.NewRouter()
	r.Handle(pattern, h).Methods("GET")
	http.Handle("/", r)

	log.Fatal(http.ListenAndServe(listen, r))
}
