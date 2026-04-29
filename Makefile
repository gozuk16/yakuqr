BINARY := yakuqr
CMD := ./cmd/yakuqr

.PHONY: build test lint clean gen-testdata

build:
	go build -o $(BINARY) $(CMD)

win:
	GOOS=windows GOARCH=amd64 go build -o windows-amd64/$(BINARY).exe $(CMD)

test:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)

gen-testdata:
	go run ./tools/gen-testdata-qr/
