package main

import (
	"bytes"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/hashicorp/consul/api"
	log "github.com/hashicorp/go-hclog"
	router "github.com/nicholasjackson/consul-connect-router"
)

var cr *router.Router

// Enables capability to run the router in AWS lambda
func main() {

	logger := log.Default()
	logger.Info("Starting Connect Router for AWS Lambda v0.1.2")

	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		logger.SetLevel(log.Debug)
	case "trace":
		logger.SetLevel(log.Trace)
	}

	config := api.DefaultConfig()
	config.Address = os.Getenv("CONSUL_ADDR")

	// Create a Consul API client
	consulClient, err := api.NewClient(config)
	if err != nil {
		logger.Error("Unable to create consul client", "error", err)
		return
	}

	upstreams := strings.Split(os.Getenv("UPSTREAMS"), ",")

	// Create and start the router
	cr, err = router.NewRouter(consulClient, logger, "", upstreams)
	if err != nil {
		logger.Error("Unable to create router", "error", err)
		return
	}

	err = cr.Run()
	if err != nil {
		logger.Error("Unable to start router", err)
	}

	lambda.Start(Handler)
}

type Request struct {
	ID    float64 `json:"id"`
	Value string  `json:"value"`
}

type Response struct {
	Message string `json:"message"`
	Ok      bool   `json:"ok"`
}

// Handler is your Lambda function handler
// It uses Amazon API Gateway request/responses provided by the aws-lambda-go/events package,
// However you could use other event sources (S3, Kinesis etc), or JSON-decoded primitive types such as 'string'.
func Handler(r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// build the request for the router
	req, err := http.NewRequest(r.HTTPMethod, "http://localhost"+r.Path, bytes.NewReader([]byte(r.Body)))
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
		}, err
	}

	pr := events.APIGatewayProxyResponse{}
	rw := &LambdaResponseWriter{&pr}

	cr.Handler(rw, req)

	return pr, nil
}

type LambdaResponseWriter struct {
	pr *events.APIGatewayProxyResponse
}

func (l *LambdaResponseWriter) Header() http.Header {
	return http.Header{}
}

func (l *LambdaResponseWriter) Write(data []byte) (int, error) {
	l.pr.Body = string(data)
	return len(data), nil
}

func (l *LambdaResponseWriter) WriteHeader(statusCode int) {
	l.pr.StatusCode = statusCode
}
