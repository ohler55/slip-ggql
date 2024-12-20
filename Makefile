
all: build

clean:
	rm *.so

lint:
	golangci-lint run

build:
	go mod tidy
	go build -buildmode=plugin -o ggql.so *.go

test: lint
	go test -coverprofile=cov.out ./...

.PHONY: all build
