package main

import (
	"github.com/gorilla/mux"
	"github.com/tilezen/xonacatl"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// LayersHandler proxies requests to an origin server and filters the response layers.
//
// It does this by matching the request against a given route pattern, and proxies that to the origin using the httpClient. It adds custom headers to that request, but strips out any header keys matching do_not_forward_headers.
type LayersHandler struct {
	origin                 *url.URL
	route                  *mux.Route
	custom_headers         *http.Header
	do_not_forward_headers []*regexp.Regexp
	httpClient             *http.Client
}

// copyAll is a simple implementation of xonacatl.LayerCopier which copies the whole response back to the client. This is useful when the server receives a request for a format it does not understand, or a request for the "all" layer, and allows it to act as a pure proxy in that case.
type copyAll struct{}

func (_ *copyAll) CopyLayers(reader io.Reader, writer io.Writer) error {
	_, err := io.Copy(writer, reader)
	return err
}

// forwardHeader returns true if the header should be forwarded to the origin.
//
// It figures that out by looking at whether header key matches any of the regular expressions in the "do not forward" list.
func (h *LayersHandler) forwardHeader(k string) bool {
	for _, re := range h.do_not_forward_headers {
		if re.MatchString(k) {
			return false
		}
	}
	return true
}

// parseRequestPath parses the request path to extract the set of layers and format of the request, as well as forming the origin request path from the variables in the route pattern.
func (h *LayersHandler) parseRequestPath(req *http.Request) (map[string]bool, string, *url.URL, error) {
	var request_layers, format string
	var pairs []string

	for k, v := range mux.Vars(req) {
		// override the layers, save the old value
		if k == "layers" {
			request_layers = v
			v = "all"

		} else if k == "fmt" {
			format = v
		}

		pairs = append(pairs, k, v)
	}

	origin_path, err := h.route.URLPath(pairs...)

	layers := make(map[string]bool)
	for _, l := range strings.Split(request_layers, ",") {
		layers[l] = true
	}

	return layers, format, origin_path, err
}

// makeProxyRequest makes a proxy request using the layers HTTP client.
//
// Note that the request's ParseForm() must have been called before this point. It is not called here so that the error can be handled separately (i.e: as a bad request, not internal server error).
func (h *LayersHandler) makeProxyRequest(origin_path string, req *http.Request) (*http.Response, error) {
	origin_url := *h.origin
	origin_url.Path = origin_path
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
		return nil, err
	}

	for k, v := range req.Header {
		if h.forwardHeader(k) {
			new_req.Header[k] = v
		}
	}

	// if there's a custom header, add it
	if h.custom_headers != nil {
		for k, vs := range *h.custom_headers {
			new_req.Header[k] = vs
		}
	}

	// delete any accept-encoding header, as the default transport for the http package will automatically and transparently gzip when possible.
	delete(new_req.Header, "Accept-Encoding")

	return h.httpClient.Do(new_req)
}

func (h *LayersHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// parse form to ensure that query parameters are available.
	err := req.ParseForm()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	layers, format, origin_path, err := h.parseRequestPath(req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := h.makeProxyRequest(origin_path.Path, req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		copyResponse(&copyAll{}, resp, rw)
		return
	}

	// we're about to modify the content, so any existing Content-Length header is very likely to be wrong.
	delete(resp.Header, "Content-Length")

	// get the appropriate copier for the layers and format
	copier := copierFor(layers, format)
	copyResponse(copier, resp, rw)
}

// copierFor returns the appropriate xonacatl.LayerCopier instance for the given set of layers and tile format.
func copierFor(layers map[string]bool, format string) (copier xonacatl.LayerCopier) {
	if layers["all"] {
		copier = &copyAll{}

	} else if format == "json" {
		copier = xonacatl.NewCopyLayers(layers)

	} else if format == "topojson" {
		copier = xonacatl.NewCopyTopoJSONLayers(layers)

	} else if format == "mvt" || format == "mvtb" {
		copier = xonacatl.NewCopyMVTLayers(layers)

	} else {
		// fall back to just copying the request as-is
		copier = &copyAll{}
	}

	return
}

// copyResponse copies an HTTP response back to the client via a xonacatl.LayerCopier, which may alter the body contents.
func copyResponse(copier xonacatl.LayerCopier, resp *http.Response, rw http.ResponseWriter) {
	for k, v := range resp.Header {
		rw.Header()[k] = v
	}
	rw.WriteHeader(resp.StatusCode)
	err := copier.CopyLayers(resp.Body, rw)

	// possibly can't return this to the client, as we've already written the
	// response header. a write failure at this stage also could be an error
	// writing _to_ the client.
	if err != nil {
		log.Printf("WARNING: Problem while writing response body: %s", err.Error())
	}
}
