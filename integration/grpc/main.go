package echo

import (
	"fmt"

	context "golang.org/x/net/context"
)

type EchoServiceServerImpl struct{}

func (e *EchoServiceServerImpl) Echo(ctx context.Context, in *Message) (*Message, error) {
	fmt.Println("Echo request")
	return in, nil
}
