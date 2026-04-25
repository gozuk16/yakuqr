BINARY := yakuqr
CMD := ./cmd/yakuqr

.PHONY: build test lint clean

build:
	go build -o $(BINARY) $(CMD)

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)
