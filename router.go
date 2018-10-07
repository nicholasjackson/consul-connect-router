package router

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/eapache/go-resiliency/retrier"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/connect"
	log "github.com/hashicorp/go-hclog"
)

// Router is an instance of a Consul Connect Router
type Router struct {
	consulClient          *api.Client
	httpClient            HTTPClient
	upstreams             Upstreams
	logger                log.Logger
	service               ConnectService
	bindAddress           string
	server                *http.Server
	httpClientFactory     func(ConnectService) HTTPClient
	connectServiceFactory func(name string) (ConnectService, error)
	registerService       func(*api.AgentServiceRegistration)
}

// NewRouter creates a new instance of the Router
func NewRouter(c *api.Client, l log.Logger, bind string, upstreams []string) (*Router, error) {
	r := &Router{
		consulClient:      c,
		logger:            l,
		bindAddress:       bind,
		httpClientFactory: buildHTTPClient,
		registerService: func(asr *api.AgentServiceRegistration) {
			c.Agent().ServiceRegister(asr)
		},
		connectServiceFactory: func(name string) (ConnectService, error) {
			return connect.NewService(name, c)
		},
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

	r.logger.Info("Starting Connect Router", "version", "0.4", "listen_addr", r.bindAddress)

	// Register the router as a Consul service
	r.registerService(&api.AgentServiceRegistration{Name: "connect-router"})

	// Create an instance representing this service. "my-service" is the
	// name of _this_ service. The service should be cleaned up via Close.
	r.service, err = r.connectServiceFactory("connect-router")
	if err != nil {
		return err
	}

	<-r.service.ReadyWait()

	// Get an HTTP client
	r.httpClient = buildHTTPClient(r.service)

	return nil
}

// ListenAndServe starts the router HTTP server
func (r *Router) ListenAndServe() error {
	// Set the handlers
	http.HandleFunc("/", r.Handler)

	// Setup the HTTP server
	r.server = &http.Server{}
	r.server.Addr = r.bindAddress

	err := r.server.ListenAndServe()
	if err != nil {
		return err
	}

	return nil
}

// Stop the router and cancel the http server
func (r *Router) Stop(ctx context.Context) {
	r.server.Shutdown(ctx)
}

// Handler defines the HTTP request handler for the router
func (r *Router) Handler(rw http.ResponseWriter, req *http.Request) {
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

	query := req.URL.RawQuery

	uri := "https://" + us.Service + ".service.consul" + path

	r.logger.Debug("Processing request", "uri", uri, "method", req.Method, "protocol", req.Proto)

	proxyReq, err := http.NewRequest(req.Method, uri, req.Body)
	if err != nil {
		r.logger.Error("Unable to create proxy request", "error", err)
		http.Error(rw, "Upable to create proxy request", http.StatusInternalServerError)
		return
	}

	proxyReq.Header.Set("Host", req.Host)
	proxyReq.Header.Set("X-Forwarded-For", req.RemoteAddr)
	proxyReq.URL.RawQuery = query

	for header, values := range req.Header {
		for _, value := range values {
			r.logger.Debug("Set request header", "header", header, "value", value)
			proxyReq.Header.Add(header, value)
		}
	}

	r.logger.Info("Attempting to request from upstream", "upstream", us.Service, "uri", path, "query", query, "method", proxyReq.Method, "protocol", proxyReq.Proto)

	var resp *http.Response

	// retry the request 3 times with a backoff
	retry := retrier.New(retrier.ConstantBackoff(3, 200*time.Millisecond), nil)
	err = retry.Run(func() error {
		var localError error
		resp, localError = r.httpClient.Do(proxyReq)
		if localError != nil {
			r.logger.Error("Unable to contact upstream", "error", localError)
			return localError
		}

		return nil
	})

	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	// set the response headers
	for header, values := range resp.Header {
		for _, value := range values {
			r.logger.Debug("Set response header", "header", header, "value", value)
			rw.Header().Add(header, value)
		}
	}

	rw.WriteHeader(resp.StatusCode)

	io.Copy(rw, resp.Body)
}

func buildHTTPClient(s ConnectService) HTTPClient {
	t := &http.Transport{
		TLSHandshakeTimeout: 20 * time.Second,
		MaxIdleConns:        20,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     120 * time.Second,

		DialTLS: s.HTTPDialTLS,
	}

	return &http.Client{
		Transport: t,
		Timeout:   10 * time.Second,
	}
}
