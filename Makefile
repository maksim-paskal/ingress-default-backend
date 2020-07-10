.PHONY: test
test:
	@./scripts/test.sh
.PHONY: build
build:
	@./scripts/validate-license.sh
	docker build .