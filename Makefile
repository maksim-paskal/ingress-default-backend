test:
	@./scripts/validate-license.sh
	go mod tidy
	go fmt ./cmd
	go vet ./cmd
	go mod tidy
	go test -race ./cmd
	golangci-lint run -v
test-k8s:
	kubectl apply --dry-run=client --validate -f deployment.yaml
run:
	go run -race -v ./cmd -http.listen=:8080
build-goreleaser:
	goreleaser build --rm-dist --snapshot
	mv ./dist/ingress-default-backend_linux_amd64/ingress-default-backend ingress-default-backend
build:
	make build-goreleaser
	docker build --pull . -t paskalmaksim/ingress-default-backend:dev
attack:
	echo "GET http://127.0.0.1:8080/" | vegeta attack -duration=120s | vegeta report
heap:
	go tool pprof -http=127.0.0.1:8081 http://127.0.0.1:8080/debug/pprof/heap