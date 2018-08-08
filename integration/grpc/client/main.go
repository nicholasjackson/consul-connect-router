package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/connect"
	echo "github.com/nicholasjackson/consul-connect-router/integration/grpc"
	"google.golang.org/grpc"
)

func main() {
	runSDK()
}

func runSDK() {
	config := api.DefaultConfig()
	config.Address = "localhost:8500"

	// Create a Consul API client
	consulClient, err := api.NewClient(config)
	if err != nil {
		fmt.Println("Unable to create consul client", "error", err)
		return
	}

	service, err := connect.NewService("grpc-client-sdk", consulClient)
	if err != nil {
		fmt.Println("Unable to create service", "error", err)
		return
	}
	defer service.Close()

	wg := sync.WaitGroup{}
	wg.Add(10)

	conn, err := service.GRPCDial("grpc-service.service.consul", grpc.WithInsecure())
	if err != nil {
		fmt.Println("Unable to create dialer", "error", err)
		return
	}
	defer conn.Close()

	// simulate multiple simultaneous request from this service
	for i := 0; i < 11; i++ {
		go func(i int, wg *sync.WaitGroup) {

			client := echo.NewEchoServiceClient(conn)
			defer wg.Done()

			st := time.Now()

			response, err := client.Echo(context.Background(), &echo.Message{Data: "abc"})
			if err != nil {
				fmt.Println(err)
				return
			}

			time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

			fmt.Println("Server responded:", i, response.Data, "in", time.Now().Sub(st))
		}(i, &wg)
	}

	wg.Wait()
}

func runThroughProxy() {
	conn, err := grpc.Dial("localhost:9011", grpc.WithInsecure())
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	client := echo.NewEchoServiceClient(conn)

	response, err := client.Echo(context.Background(), &echo.Message{Data: "abc"})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Server responded:", response.Data)
}
