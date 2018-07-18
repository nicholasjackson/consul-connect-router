package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/connect"
	log "github.com/hashicorp/go-hclog"
	flag "github.com/spf13/pflag"
)

var consulAddr = flag.String("consul_addr", "http://127.0.0.1:8500", "Address of Consul agent")
var upstream = flag.StringSlice("upstream", nil, "define upstreams with [service]#[path] i.e http-echo#/")
var listen = flag.String("listen", ":8181", "listen address i.e localhost:8181")

var consulClient *api.Client
var httpClient *http.Client
var upstreams Upstreams
var logger log.Logger
var service *connect.Service

func main() {
	flag.Parse()

	logger = log.Default()
	logger.Info("Starting Connect Router", "version", "0.1")

	config := api.DefaultConfig()
	config.Address = *consulAddr

	var err error
	upstreams, err = NewUpstreams(*upstream)
	if err != nil {
		logger.Error("Unable to parse upstreams", "error", err)
	}

	logger.Info("Upstreams configuration", "upstreams", upstreams)

	// Create a Consul API client
	consulClient, err = api.NewClient(config)
	if err != nil {
		logger.Error("Unable to create consul client", "error", err)
		return
	}

	consulClient.Agent().ServiceRegister(&api.AgentServiceRegistration{Name: "connect-router"})
	// Create an instance representing this service. "my-service" is the
	// name of _this_ service. The service should be cleaned up via Close.
	service, err = connect.NewService("connect-router", consulClient)
	if err != nil {
		logger.Error("Unable to create connect service", "error", err)
		return
	}
	defer service.Close()

	// Get an HTTP client
	httpClient = service.HTTPClient()
	http.HandleFunc("/", handler)

	// setup the server
	s := http.Server{}
	s.Addr = *listen

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		logger.Info("Received termination signal, shutting down", "signal", sig)

		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		s.Shutdown(ctx)
	}()

	err = s.ListenAndServe()
	if err != nil {
		logger.Error("Error shutting down server", "error", err)
	}
}

func handler(rw http.ResponseWriter, r *http.Request) {
	//find the upstream
	us := upstreams.FindUpstream(r.URL.Path)
	if us == nil {
		logger.Error("No upstream defined", "path", r.URL.Path)
		http.Error(rw, "No upstream defined for path", http.StatusNotFound)
		return
	}

	// strip the prefix from the router
	// TODO: make this optional
	path := strings.TrimPrefix(r.URL.Path, us.Path)

	// if the path does not start with a / add one
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	uri := "https://" + us.Host + ".service.consul" + path

	proxyReq, err := http.NewRequest(r.Method, uri, r.Body)
	if err != nil {
		logger.Error("Unable to create proxy request", "error", err)
		http.Error(rw, "Upable to create proxy request", http.StatusInternalServerError)
		return
	}

	proxyReq.Header.Set("Host", r.Host)
	proxyReq.Header.Set("X-Forwarded-For", r.RemoteAddr)

	for header, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(header, value)
		}
	}

	logger.Info("Attempting to request from upstream", "upstream", us.Host, "path", path)

	resp, err := httpClient.Do(proxyReq)
	if err != nil {
		logger.Error("Unable to contact upstream", "error", err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	// set the response headers
	for header, values := range resp.Header {
		for _, value := range values {
			rw.Header().Set(header, value)
		}
	}

	rw.WriteHeader(http.StatusOK)
	io.Copy(rw, resp.Body)
}
