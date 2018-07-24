package router

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/connect"
	log "github.com/hashicorp/go-hclog"
)

// Router is an instance of a Consul Connect Router
type Router struct {
	consulClient *api.Client
	httpClient   *http.Client
	upstreams    Upstreams
	logger       log.Logger
	service      *connect.Service
	bindAddress  string
	server       *http.Server
}

// NewRouter creates a new instance of the Router
func NewRouter(c *api.Client, l log.Logger, bind string, upstreams []string) (*Router, error) {
	r := &Router{
		consulClient: c,
		logger:       l,
		bindAddress:  bind,
	}

	var err error
	r.upstreams, err = NewUpstreams(upstreams)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse upstreams: %s", err)
	}

	return r, nil
}

// Run starts the router and listens on the defined address
func (r *Router) Run() error {
	var err error

	r.logger.Info("Starting Connect Router", "version", "0.2")
	r.consulClient.Agent().ServiceRegister(&api.AgentServiceRegistration{Name: "connect-router"})

	// Create an instance representing this service. "my-service" is the
	// name of _this_ service. The service should be cleaned up via Close.
	r.service, err = connect.NewService("connect-router", r.consulClient)
	if err != nil {
		return err
	}
	defer r.service.Close()

	// Get an HTTP client
	r.httpClient = r.service.HTTPClient()

	// Set the handlers
	http.HandleFunc("/", r.handler)

	// setup the server
	r.server = &http.Server{}
	r.server.Addr = r.bindAddress

	err = r.server.ListenAndServe()
	if err != nil {
		return err
	}

	return nil
}

// Stop the router and cancel the http server
func (r *Router) Stop(ctx context.Context) {
	r.server.Shutdown(ctx)
}

func (r *Router) handler(rw http.ResponseWriter, req *http.Request) {
	//find the upstream
	us := r.upstreams.FindUpstream(req.URL.Path)
	if us == nil {
		r.logger.Error("No upstream defined", "path", req.URL.Path)
		http.Error(rw, "No upstream defined for path", http.StatusNotFound)
		return
	}

	// strip the prefix from the router
	// TODO: make this optional
	path := strings.TrimPrefix(req.URL.Path, us.Path)

	// if the path does not start with a / add one
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	uri := "https://" + us.Host + ".service.consul" + path

	proxyReq, err := http.NewRequest(req.Method, uri, req.Body)
	if err != nil {
		r.logger.Error("Unable to create proxy request", "error", err)
		http.Error(rw, "Upable to create proxy request", http.StatusInternalServerError)
		return
	}

	proxyReq.Header.Set("Host", req.Host)
	proxyReq.Header.Set("X-Forwarded-For", req.RemoteAddr)

	for header, values := range req.Header {
		for _, value := range values {
			proxyReq.Header.Add(header, value)
		}
	}

	r.logger.Info("Attempting to request from upstream", "upstream", us.Host, "path", path)

	resp, err := r.httpClient.Do(proxyReq)
	if err != nil {
		r.logger.Error("Unable to contact upstream", "error", err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	// set the response headers
	for header, values := range resp.Header {
		for _, value := range values {
			rw.Header().Add(header, value)
		}
	}

	rw.WriteHeader(resp.StatusCode)
	io.Copy(rw, resp.Body)
}
