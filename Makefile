test:
	@./scripts/validate-license.sh
	go mod tidy
	go fmt ./cmd
	go mod tidy
	go test -race ./cmd
	golangci-lint run -v
run:
	go run -race -v ./cmd -http.listen=:8080
build:
	docker build . -t paskalmaksim/ingress-default-backend:dev
attack:
	echo "GET http://127.0.0.1:8080/" | vegeta attack -duration=120s | vegeta report
heap:
	go tool pprof -http=127.0.0.1:8081 http://127.0.0.1:8080/debug/pprof/heap