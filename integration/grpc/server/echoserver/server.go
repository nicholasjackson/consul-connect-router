package echoserver

import (
	"fmt"

	echo "github.com/nicholasjackson/consul-connect-router/integration/grpc"
	context "golang.org/x/net/context"
)

type EchoServiceServerImpl struct {
	ID string
}

func (e *EchoServiceServerImpl) Echo(ctx context.Context, in *echo.Message) (*echo.Message, error) {
	fmt.Println("Server id", e.ID, "Echo request", in.Data)
	return in, nil
}
