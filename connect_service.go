package router

import (
	"net"

	"github.com/stretchr/testify/mock"
)

// ConnectService defines the interface methods uses on the connect service api
type ConnectService interface {
	Close() error
	ReadyWait() <-chan struct{}
	HTTPDialTLS(network, addr string) (net.Conn, error)
}

type MockConnectService struct {
	mock.Mock
}

func (m *MockConnectService) Close() error {
	args := m.Mock.Called()

	return args.Error(0)
}

func (m *MockConnectService) ReadyWait() <-chan struct{} {
	args := m.Mock.Called()

	return args.Get(0).(chan struct{})
}

func (m *MockConnectService) HTTPDialTLS(network, addr string) (net.Conn, error) {
	args := m.Called(network, addr)

	return args.Get(0).(net.Conn), args.Error(1)
}
