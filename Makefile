BINARY=cfst-client

.PHONY: build

build:
    go build -o $(BINARY) ./cmd/main.go

clean:
    rm -f $(BINARY)
