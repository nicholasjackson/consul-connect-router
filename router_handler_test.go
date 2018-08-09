package router

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var mockHTTPClient *MockHTTPClient
var httpResponse *http.Response
var mockConnectService *MockConnectService

func setupRouterTests(t *testing.T) *Router {
	httpResponse = &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader([]byte("testbody"))),
	}

	mockHTTPClient = &MockHTTPClient{}
	mockHTTPClient.On("Do", mock.Anything).Return(httpResponse, nil)

	cha := make(chan struct{}, 1)
	cha <- struct{}{}
	mockConnectService = &MockConnectService{}
	mockConnectService.On("ReadyWait").Return(cha)
	mockConnectService.On("Close").Return(nil)

	r := &Router{
		httpClient: mockHTTPClient,
		logger:     log.Default(),
		upstreams:  Upstreams{},
	}

	return r
}

func TestHandlerReturnsErrorWhenUpstreamNotFound(t *testing.T) {
	rec := setupRouterTests(t)
	rw := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)

	rec.Handler(rw, r)

	assert.Equal(t, rw.Code, http.StatusNotFound, "Should have returned not found")
}

func TestHandlerTrimsPrefixFromRequest(t *testing.T) {
	rec := setupRouterTests(t)
	rec.upstreams = append(
		rec.upstreams,
		Upstream{
			Service: "test",
			Path:    "/test",
		})
	rw := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)

	rec.Handler(rw, r)

	mockHTTPClient.AssertCalled(t, "Do", mock.Anything)
	req := mockHTTPClient.Calls[0].Arguments.Get(0).(*http.Request)

	assert.Equal(t, "/", req.URL.Path, "Should have stripped the prefix")
}

func TestHandlerBuildsCorrectURI(t *testing.T) {
	rec := setupRouterTests(t)
	rec.upstreams = append(
		rec.upstreams,
		Upstream{
			Service: "test",
			Path:    "/test",
		})
	rw := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)

	rec.Handler(rw, r)

	mockHTTPClient.AssertCalled(t, "Do", mock.Anything)
	req := mockHTTPClient.Calls[0].Arguments.Get(0).(*http.Request)

	assert.Equal(t, "https://test.service.consul/", req.URL.String(), "Should have stripped the prefix")
}

func TestHandlerSetsCustomRequestHeaders(t *testing.T) {
	rec := setupRouterTests(t)
	rec.upstreams = append(
		rec.upstreams,
		Upstream{
			Service: "test",
			Path:    "/test",
		})
	rw := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	r.Host = "testhost"
	r.RemoteAddr = "192.168.1.1"

	rec.Handler(rw, r)

	mockHTTPClient.AssertCalled(t, "Do", mock.Anything)
	req := mockHTTPClient.Calls[0].Arguments.Get(0).(*http.Request)

	assert.Equal(t, r.Host, req.Header.Get("Host"), "Should have set host header")
	assert.Equal(t, r.RemoteAddr, req.Header.Get("X-Forwarded-For"), "Should have set forwarded for")
}

func TestHandlerCopiesHeadersFromRequest(t *testing.T) {
	rec := setupRouterTests(t)
	rec.upstreams = append(
		rec.upstreams,
		Upstream{
			Service: "test",
			Path:    "/test",
		})
	rw := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	r.Header.Set("test1", "abc")
	r.Header.Set("test2", "def")

	rec.Handler(rw, r)

	mockHTTPClient.AssertCalled(t, "Do", mock.Anything)
	req := mockHTTPClient.Calls[0].Arguments.Get(0).(*http.Request)

	assert.Equal(t, "abc", req.Header.Get("test1"), "Should have set copied test1 header")
	assert.Equal(t, "def", req.Header.Get("test2"), "Should have set copied test2 header")
}

func TestHandlerCopiesHeadersFromResponse(t *testing.T) {
	rec := setupRouterTests(t)
	rec.upstreams = append(
		rec.upstreams,
		Upstream{
			Service: "test",
			Path:    "/test",
		})
	rw := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	httpResponse.Header = http.Header{}
	httpResponse.Header.Set("resp1", "abc")
	httpResponse.Header.Set("resp2", "def")

	rec.Handler(rw, r)

	mockHTTPClient.AssertCalled(t, "Do", mock.Anything)

	assert.Equal(t, "abc", rw.Header().Get("resp1"), "Should have set copied resp1 header")
	assert.Equal(t, "def", rw.Header().Get("resp2"), "Should have set copied resp2 header")
}

func TestHandlerCopiesBodyFromResponse(t *testing.T) {
	rec := setupRouterTests(t)
	rec.upstreams = append(
		rec.upstreams,
		Upstream{
			Service: "test",
			Path:    "/test",
		})
	rw := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)

	rec.Handler(rw, r)

	mockHTTPClient.AssertCalled(t, "Do", mock.Anything)

	assert.Equal(t, "testbody", rw.Body.String(), "Should have set copied response body")
}

func TestHandlerSetsStatusCodeFromResponse(t *testing.T) {
	rec := setupRouterTests(t)
	rec.upstreams = append(
		rec.upstreams,
		Upstream{
			Service: "test",
			Path:    "/test",
		})
	rw := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	httpResponse.StatusCode = http.StatusTeapot

	rec.Handler(rw, r)

	mockHTTPClient.AssertCalled(t, "Do", mock.Anything)

	assert.Equal(t, http.StatusTeapot, rw.Code, "Should have set status code from response")
}
