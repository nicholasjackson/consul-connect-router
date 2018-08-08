package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/colors"
	"github.com/DATA-DOG/godog/gherkin"
	"github.com/hashicorp/consul/api"
	log "github.com/hashicorp/go-hclog"
	router "github.com/nicholasjackson/consul-connect-router"
	echo "github.com/nicholasjackson/consul-connect-router/integration/grpc"
	"google.golang.org/grpc"
)

var opt = godog.Options{Output: colors.Colored(os.Stdout)}
var consulProc *exec.Cmd
var consulProxy *exec.Cmd
var response *echo.Message

const (
	proxyAddr  = "localhost:8443"
	routerAddr = "localhost:9090"
	grpcAddr   = "localhost:9999"
	consulAddr = "localhost:8510"
)

func init() {
	godog.BindFlags("godog.", flag.CommandLine, &opt)
}

func TestMain(m *testing.M) {
	flag.Parse()
	opt.Paths = flag.Args()

	status := godog.RunWithOptions("godogs", func(s *godog.Suite) {
		FeatureContext(s)
	}, opt)

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}

func theGRPCEchoServerAndProxyIsRunning() error {
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	echo.RegisterEchoServiceServer(grpcServer, &echo.EchoServiceServerImpl{})
	go grpcServer.Serve(lis)

	startProxy()
	return nil
}

func theRouterIsRunning() error {
	config := api.DefaultConfig()
	config.Address = consulAddr

	// Create a Consul API client
	consulClient, err := api.NewClient(config)
	if err != nil {
		return err
	}

	r, err := router.NewRouter(
		consulClient,
		log.Default(),
		routerAddr,
		[]string{
			"service=test#path=/",
		})

	if err != nil {
		return err
	}

	go r.Run()
	time.Sleep(2 * time.Second)

	return nil
}

func iSendARequestToTheRouter() error {
	conn, err := grpc.Dial(proxyAddr, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := echo.NewEchoServiceClient(conn)

	response, err = client.Echo(context.Background(), &echo.Message{Data: "abc"})
	if err != nil {
		return err
	}

	return nil
}

func theGRPCServersResponseShouldBeEchoed() error {
	if response == nil {
		return fmt.Errorf("No response from server")
	}

	if response.Data != "abc" {
		return fmt.Errorf("Server did not echo response, expected abc, got %v", response.Data)
	}

	return nil
}

func FeatureContext(s *godog.Suite) {
	s.BeforeFeature(setup)
	s.AfterFeature(teardown)

	s.Step(`^the gRPC echo server and proxy is running$`, theGRPCEchoServerAndProxyIsRunning)
	s.Step(`^the router is running$`, theRouterIsRunning)
	s.Step(`^I send a request to the router$`, iSendARequestToTheRouter)
	s.Step(`^the gRPC servers response should be echoed$`, theGRPCServersResponseShouldBeEchoed)
}

func setup(f *gherkin.Feature) {
	startConsul()
}

func teardown(f *gherkin.Feature) {
	// kill consul
	err := consulProc.Process.Kill()
	if err != nil {
		panic(err)
	}

	// kill proxy
	err = consulProxy.Process.Kill()
	if err != nil {
		panic(err)
	}
}

// startConsul in dev mode
func startConsul() {
	//var out bytes.Buffer
	//var stderr bytes.Buffer
	consulProc = exec.Command(
		"/usr/local/bin/consul",
		"agent",
		"-dev",
		"-config-file", "./consul.hcl",
	)

	consulProc.Stdout = os.Stdout
	consulProc.Stderr = os.Stderr

	go func() {
		err := consulProc.Run()
		if err != nil {
			//fmt.Println(stderr.String())
			panic(err)
		}
	}()

	time.Sleep(2 * time.Second)
}

// startConsul in dev mode
func startProxy() {
	//var out bytes.Buffer
	//var stderr bytes.Buffer
	consulProxy = exec.Command(
		"/usr/local/bin/consul",
		"connect",
		"proxy",
		"-service", "grpctest",
		"-service-addr", grpcAddr,
		"-log-level", "debug",
		"-listen", proxyAddr,
		"-http-addr", consulAddr,
		"-register",
	)

	consulProxy.Stdout = os.Stdout
	consulProxy.Stderr = os.Stderr

	go func() {
		err := consulProxy.Run()
		if err != nil {
			//fmt.Println(stderr.String())
			panic(err)
		}
	}()

	time.Sleep(2 * time.Second)
}
