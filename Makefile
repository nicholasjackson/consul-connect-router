run:
	GODEBUG=http2debug=1 go run main.go upstream.go --upstream "http-echo#/http" --upstream "socat-echo#/socat" --upstream "http-api#/api"

build:
	goreleaser --snapshot --rm-dist --skip-publish

run_proxy_http:
	http-echo -listen 127.0.0.1:8087 -text hello 2>"/tmp/http_echo.out" &
	curl -s -X PUT -d @"$(shell pwd)/integration/service.json" "http://127.0.0.1:8500/v1/agent/service/register" | jq

run_proxy_socat:
	socat -v tcp-l:8080,bind=127.0.0.1,fork exec:"/bin/cat" 2>"/tmp/socat-echo.out" &
	curl -s -X PUT -d @"$(shell pwd)/integration/service2.json" "http://127.0.0.1:8500/v1/agent/service/register" | jq

run_test:
	curl -s -X PUT -d @"$(shell pwd)/integration/service3.json" "http://127.0.0.1:8500/v1/agent/service/register" | jq

deregister_all:
	pkill http-echo || true
	pkill socat || true
	curl -XPUT localhost:8500/v1/agent/service/deregister/http-echo
	curl -XPUT localhost:8500/v1/agent/service/deregister/socat-echo
