tag=dev
image=paskalmaksim/ingress-default-backend:$(tag)

test:
	@./scripts/validate-license.sh
	go mod tidy
	go fmt ./cmd
	go vet ./cmd
	go mod tidy
	go test -race ./cmd
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run -v
test-k8s:
	kubectl apply --dry-run=client --validate -f deployment.yaml
run:
	go run -race -v ./cmd -http.listen=:8080
build-goreleaser:
	git tag -d `git tag -l "helm-chart-*"`
	go run github.com/goreleaser/goreleaser@latest build --rm-dist --snapshot
	mv ./dist/ingress-default-backend_linux_amd64/ingress-default-backend ingress-default-backend
build:
	make build-goreleaser
	docker build --pull . -t $(image)
attack:
	echo "GET http://127.0.0.1:8080/" | vegeta attack -duration=120s | vegeta report
heap:
	go tool pprof -http=127.0.0.1:8081 http://127.0.0.1:8080/debug/pprof/heap
scan:
	@trivy image \
	-ignore-unfixed --no-progress --severity HIGH,CRITICAL \
	$(image)