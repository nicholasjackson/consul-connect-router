package router

import (
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
)

func TestRunRegistersService(t *testing.T) {
	var registerParams *api.AgentServiceRegistration

	r := setupRouterTests(t)
	r.registerService = func(asr *api.AgentServiceRegistration) {
		registerParams = asr
	}
	r.connectServiceFactory = func(name string) (ConnectService, error) {
		return mockConnectService, nil
	}

	err := r.Run()

	assert.NoError(t, err)
	assert.Equal(t, "connect-router", registerParams.Name)
}
