run:
	go run main.go upstream.go --upstream "db#/"

build:
	goreleaser --snapshot --rm-dist --skip-publish

run_proxy:
	curl -s -X PUT -d @"$(shell pwd)/integration/service.json" "http://127.0.0.1:8500/v1/agent/service/register" | jq
