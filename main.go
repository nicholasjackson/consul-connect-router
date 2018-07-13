package main

import (
	"io"
	"net/http"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/connect"
	log "github.com/hashicorp/go-hclog"
	flag "github.com/spf13/pflag"
)

var consulAddr = flag.String("consul_addr", "http://127.0.0.1:8500", "Address of Consul agent")
var upstream = flag.StringSlice("upstream", nil, "define upstreams with [service]#[path] i.e http-echo#/")

var httpClient *http.Client
var upstreams Upstreams
var logger log.Logger

func main() {
	flag.Parse()

	logger = log.Default()
	logger.Info("Starting Connect Router")

	config := api.DefaultConfig()
	config.Address = *consulAddr

	var err error
	upstreams, err = NewUpstreams(*upstream)
	if err != nil {
		logger.Error("Unable to parse upstreams", "error", err)
	}

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

func handler(rw http.ResponseWriter, r *http.Request) {
	//find the upstream
	us := upstreams.FindUpstream(r.URL.Path)
	if us == nil {
		logger.Error("No upstream defined", "path", r.URL.Path)
		http.Error(rw, "No upstream defined for path", http.StatusNotFound)
		return
	}

	resp, err := httpClient.Get("https://" + us.Host + ".service.consul")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()
	io.Copy(rw, resp.Body)
}