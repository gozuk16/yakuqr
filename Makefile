BINARY := yakuqr
CMD := ./cmd/yakuqr

.PHONY: build test lint clean gen-testdata

build:
	go build -o $(BINARY) $(CMD)

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)

gen-testdata:
	go run ./tools/gen-testdata-qr/
