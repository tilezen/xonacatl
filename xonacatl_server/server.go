package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/NYTimes/gziphandler"
	"github.com/gorilla/mux"
	"github.com/namsral/flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
)

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
			http_client:            &http.Client{},
		}

		gzipped := gziphandler.GzipHandler(h)

		r.Handle(pattern, gzipped).Methods("GET")
	}

	if len(healthcheck) > 0 {
		r.HandleFunc(healthcheck, getHealth).Methods("GET")
	}
	http.Handle("/", r)

	log.Fatal(http.ListenAndServe(listen, r))
}
