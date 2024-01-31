.PHONY: generate
generate:
	@go generate ./...

.PHONY: acndev
acndev: 
	mkdir -p ./bin
		CGO_ENABLED=0 go build -o ./bin/acndev ./cmd/
