package main

import (
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/connect"
	log "github.com/hashicorp/go-hclog"
	flag "github.com/spf13/pflag"
)

var consulAddr = flag.String("consul_addr", "http://127.0.0.1:8500", "")
var upstream = flag.StringSlice("upstream", nil, "define upstreams with [service]#[path] i.e http-echo#/")

var httpClient *http.Client
var upstreams []Upstream
var logger log.Logger

// Upstream defines a struct to encapsulate upstream info
type Upstream struct {
	Host string
	Path string
}

func main() {
	flag.Parse()

	logger = log.Default()
	logger.Info("Starting Connect Router")

	config := api.DefaultConfig()
	config.Address = *consulAddr

	upstreams = parseUpstreams()
	logger.Info("Upstreams configuration", "upstreams", upstreams)

	// Create a Consul API client
	client, err := api.NewClient(config)
	if err != nil {
		logger.Error("Unable to create consul client", "error", err)
		return
	}

	// Create an instance representing this service. "my-service" is the
	// name of _this_ service. The service should be cleaned up via Close.
	svc, err := connect.NewService("connect-router", client)
	if err != nil {
		logger.Error("Unable to create connect service", "error", err)
		return
	}
	defer svc.Close()

	// Get an HTTP client
	httpClient = svc.HTTPClient()
	http.HandleFunc("/", handler)

	http.ListenAndServe(":8181", nil)
}

func parseUpstreams() []Upstream {
	us := make([]Upstream, 0)

	for _, v := range *upstream {
		parts := strings.Split(v, "#")
		us = append(us, Upstream{
			Host: parts[0],
			Path: parts[1],
		})
	}

	return us
}

func handler(rw http.ResponseWriter, r *http.Request) {
	//find the upstream
	for _, us := range upstreams {
		if strings.HasPrefix(r.URL.Path, us.Path) {
			resp, err := httpClient.Get("https://" + us.Host + ".service.consul")
			if err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}

			defer resp.Body.Close()
			io.Copy(rw, resp.Body)
			return
		}
	}

	logger.Error("No upstream defined", "path", r.URL.Path)
	http.Error(rw, "No upstream defined for path", http.StatusNotFound)
}
