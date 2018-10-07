package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/consul/api"
	log "github.com/hashicorp/go-hclog"
	router "github.com/nicholasjackson/consul-connect-router"
	flag "github.com/spf13/pflag"
)

var consulAddr = flag.String("consul_addr", "http://127.0.0.1:8500", "Address of Consul agent")
var upstream = flag.StringSlice("upstream", nil, "define upstreams with service=[service]#path=[path] i.e service=http-echo#path=/")
var listen = flag.String("listen", ":8181", "listen address i.e localhost:8181")
var logLevel = flag.String("log_level", "info", "log level, info, debug, trace")

var logger log.Logger

func main() {
	flag.Parse()

	logger = log.Default()

	switch *logLevel {
	case "debug":
		logger.SetLevel(log.Debug)
	case "trace":
		logger.SetLevel(log.Trace)
	}

	config := api.DefaultConfig()
	config.Address = *consulAddr

	// Create a Consul API client
	consulClient, err := api.NewClient(config)
	if err != nil {
		logger.Error("Unable to create consul client", "error", err)
		return
	}

	// Create and start the router
	r, err := router.NewRouter(consulClient, logger, *listen, *upstream)
	if err != nil {
		logger.Error("Unable to create router", "error", err)
		return
	}

	// ensure the router stops cleanly when sigterm is detected
	handleSigTerm(r)

	r.Run()
	r.ListenAndServe()
}

func handleSigTerm(r *router.Router) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		logger.Info("Received termination signal, shutting down", "signal", sig)

		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		r.Stop(ctx)
	}()
}
