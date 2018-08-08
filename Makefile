run:
	GODEBUG=http2debug=2 go run ./cmd/main.go --log_level debug --upstream "service=http-tls#path=/tls" --upstream "service=http-echo#path=/http" --upstream "service=socat-echo#path=/socat" --upstream "service=http-api#path=/api"

build:
	goreleaser --snapshot --rm-dist --skip-publish

build_lambda:
	GOOS=linux go build -o ./lambda/main ./lambda/main.go

goconvey:
	goconvey -excludedDirs 'integration,vendor'

functional_test:
	cd integration && go test -v --godog.format=pretty --godog.random

functional_build_proto:
	protoc -I ./integration/grpc/ integration/grpc/echo.proto --go_out=plugins=grpc:integration/grpc

run_grpc_service:
	curl -s -X PUT -d @"$(shell pwd)/integration/grpc_service1.json" "http://127.0.0.1:8500/v1/agent/service/register" | jq
	curl -s -X PUT -d @"$(shell pwd)/integration/grpc_service2.json" "http://127.0.0.1:8500/v1/agent/service/register" | jq
	go build -o integration/grpc/server/grpc_server integration/grpc/server/main.go 
	./integration/grpc/server/grpc_server -listen "0.0.0.0:9998" -id "1" 2>"/tmp/grpc_service1.out" &
	./integration/grpc/server/grpc_server -listen "0.0.0.0:9999" -id "2" 2>"/tmp/grpc_service2.out" &

run_grpc_client:
	curl -s -X PUT -d @"$(shell pwd)/integration/grpc_client.json" "http://127.0.0.1:8500/v1/agent/service/register" | jq
	echo "Run 'go run integration/grpc/client/main.go' to run the client"

run_proxy_http:
	http-echo -listen 127.0.0.1:8087 -text hello 2>"/tmp/http_echo.out" &
	curl -s -X PUT -d @"$(shell pwd)/integration/service.json" "http://127.0.0.1:8500/v1/agent/service/register" | jq

run_proxy_socat:
	socat -v tcp-l:8080,bind=127.0.0.1,fork exec:"/bin/cat" 2>"/tmp/socat-echo.out" &
	curl -s -X PUT -d @"$(shell pwd)/integration/service2.json" "http://127.0.0.1:8500/v1/agent/service/register" | jq

run_test:
	curl -s -X PUT -d @"$(shell pwd)/integration/service3.json" "http://127.0.0.1:8500/v1/agent/service/register" | jq

run_proxy_tls:
	curl -s -X PUT -d @"$(shell pwd)/integration/service4.json" "http://127.0.0.1:8500/v1/agent/service/register" | jq
	go run ./integration/https/main.go

deregister_all:
	pkill http-echo || true
	pkill socat || true
	curl -XPUT localhost:8500/v1/agent/service/deregister/http-echo
	curl -XPUT localhost:8500/v1/agent/service/deregister/http-tls
	curl -XPUT localhost:8500/v1/agent/service/deregister/socat-echo
	curl -XPUT localhost:8500/v1/agent/service/deregister/grpc-service-1
	curl -XPUT localhost:8500/v1/agent/service/deregister/grpc-service-2
	curl -XPUT localhost:8500/v1/agent/service/deregister/grpc-client
