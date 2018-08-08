package main

import (
	"flag"
	"fmt"
	"net"

	echo "github.com/nicholasjackson/consul-connect-router/integration/grpc"
	"github.com/nicholasjackson/consul-connect-router/integration/grpc/server/echoserver"
	"google.golang.org/grpc"
)

var listen = flag.String("listen", "", "")
var serverid = flag.String("id", "", "")

func main() {
	flag.Parse()

	fmt.Println("Starting grpc server, listen address", *listen)

	lis, err := net.Listen("tcp", *listen)
	if err != nil {
		fmt.Println(err)
	}

	grpcServer := grpc.NewServer()
	echo.RegisterEchoServiceServer(grpcServer, &echoserver.EchoServiceServerImpl{ID: *serverid})
	grpcServer.Serve(lis)
}
