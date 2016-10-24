package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/namsral/flag"
	"github.com/tilezen/xonacatl"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

type LayersHandler struct {
	origin                 *url.URL
	route                  *mux.Route
	custom_headers         *http.Header
	do_not_forward_headers []*regexp.Regexp
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

		} else if k == "fmt" {
			format = v
		}

		pairs = append(pairs, k, v)
	}

	origin_path, err := h.route.URLPath(pairs...)
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}

	// parse form to ensure that query parameters are available.
	err = req.ParseForm()
	if err != nil {
		http.Error(rw, err.Error(), 400)
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

HeaderLoop:
	for k, v := range req.Header {
		for _, re := range h.do_not_forward_headers {
			if re.MatchString(k) {
				continue HeaderLoop
			}
		}

		new_req.Header[k] = v
	}
	// always request gzip from upstream, regardless of what the client asked
	// for. ask for gzip first, with fall back to identity
	new_req.Header["Accept-Encoding"] = []string{"gzip;q=1.0,identity;q=0.5"}
	// if there's a custom header, add it
	if h.custom_headers != nil {
		for k, vs := range *h.custom_headers {
			new_req.Header[k] = vs
		}
	}

	resp, err := http.DefaultClient.Do(new_req)
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}

	if resp.StatusCode != 200 {
		for k, v := range resp.Header {
			rw.Header()[k] = v
		}
		rw.WriteHeader(resp.StatusCode)
		copyAll(resp.Body, map[string]bool{}, rw)
		return
	}

	var rd io.Reader
	rd = resp.Body
	content_encoding := resp.Header["Content-Encoding"]
	if content_encoding != nil &&
		len(content_encoding) == 1 &&
		content_encoding[0] == "gzip" {
		rd, err = gzip.NewReader(rd)
		if err != nil {
			http.Error(rw, err.Error(), 500)
			return
		}
	}

	var wr io.Writer
	wr = rw
	// TODO: proper content negotiation on quality values
	req_content_encoding := "identity"
	if strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
		gz := gzip.NewWriter(wr)
		defer func() {
			err = gz.Close()
			if err != nil {
				log.Printf("Failed to flush and close GzipWriter: %s", err.Error())
			}
		}()
		wr = gz
		req_content_encoding = "gzip"
	}

	for k, v := range resp.Header {
		rw.Header()[k] = v
	}
	rw.Header()["Content-Encoding"] = []string{req_content_encoding}
	delete(rw.Header(), "Content-Length")
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

	} else if format == "topojson" {
		copy_func = xonacatl.CopyTopoJSONLayers

	} else if format == "mvt" || format == "mvtb" {
		copy_func = xonacatl.CopyMVTLayers

	} else {
		// fall back to just copying the request as-is
		copy_func = copyAll
	}

	err = copy_func(rd, layers, wr)

	// possibly can't return this to the client, as we've already written the
	// response header. a write failure at this stage also could be an error
	// writing _to_ the client.
	if err != nil {
		log.Printf("Error while writing response body: %s", err.Error())
	}
}

type headerOption struct {
	header http.Header
}

func (h *headerOption) String() string {
	var buf bytes.Buffer
	h.header.Write(&buf)
	return buf.String()
}

func (h *headerOption) Set(line string) error {
	m := make(map[string]string)
	err := json.Unmarshal([]byte(line), &m)
	if err != nil {
		return fmt.Errorf("Unable to parse value as a JSON object: %s", err.Error())
	}

	for k, v := range m {
		h.header.Set(k, v)
	}

	return nil
}

func getHealth(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(200)
}

type patternsOption struct {
	patterns map[string]*url.URL
}

func (p *patternsOption) String() string {
	return fmt.Sprintf("%#v", p.patterns)
}

func (p *patternsOption) Set(line string) error {
	m := make(map[string]string)
	err := json.Unmarshal([]byte(line), &m)
	if err != nil {
		return fmt.Errorf("Unable to parse value as a JSON object: %s", err.Error())
	}

	for k, v := range m {
		url, err := url.Parse(v)
		if err != nil {
			return fmt.Errorf("Unable to parse origin URL %#v: %s", v, err.Error())
		}

		p.patterns[k] = url
	}

	return nil
}

type regexpListOption struct {
	regexps []*regexp.Regexp
}

func (r *regexpListOption) String() string {
	return fmt.Sprintf("%#v", r.regexps)
}

func (r *regexpListOption) Set(line string) error {
	var parts []string
	err := json.Unmarshal([]byte(line), &parts)
	if err != nil {
		return fmt.Errorf("Unable to parse value as a JSON list: %s", err.Error())
	}

	for _, part := range parts {
		re, err := regexp.Compile(part)
		if err != nil {
			return fmt.Errorf("Unable to compile part #%v as regexp: %s", part, err.Error())
		}
		r.regexps = append(r.regexps, re)
	}

	return nil
}

func main() {
	var listen, healthcheck string
	custom_headers := headerOption{header: make(http.Header)}
	patterns := patternsOption{patterns: make(map[string]*url.URL)}
	do_not_forward := regexpListOption{}

	f := flag.NewFlagSetWithEnvPrefix(os.Args[0], "XONACATL", 0)
	f.Var(&patterns, "patterns", "JSON object of patterns to use when matching incoming tile requests.")
	f.StringVar(&listen, "listen", ":8080", "interface and port to listen on")
	f.String("config", "", "Config file to read values from.")
	f.Var(&custom_headers, "headers", "JSON object of extra headers to add to proxied requests.")
	f.StringVar(&healthcheck, "healthcheck", "", "A path to respond to with a blank 200 OK. Intended for use by load balancer health checks.")
	f.Var(&do_not_forward, "noforward", "List of regular expressions. If a header matches one of these, then it will not be forwarded to the origin.")
	err := f.Parse(os.Args[1:])
	if err == flag.ErrHelp {
		return
	} else if err != nil {
		log.Fatalf("Unable to parse input command line, environment or config: %s", err.Error())
	}

	if len(patterns.patterns) == 0 {
		log.Fatalf("You must provide at least one pattern to proxy.")
	}

	var headers *http.Header
	if len(custom_headers.header) > 0 {
		headers = &custom_headers.header
	}

	r := mux.NewRouter()

	for pattern, origin := range patterns.patterns {
		origin_router := mux.NewRouter()
		origin_router.NewRoute().Path(origin.Path).BuildOnly().Name("origin")

		h := &LayersHandler{
			origin:                 origin,
			route:                  origin_router.GetRoute("origin"),
			custom_headers:         headers,
			do_not_forward_headers: do_not_forward.regexps,
		}

		r.Handle(pattern, h).Methods("GET")
	}

	if len(healthcheck) > 0 {
		r.HandleFunc(healthcheck, getHealth).Methods("GET")
	}
	http.Handle("/", r)

	log.Fatal(http.ListenAndServe(listen, r))
}
