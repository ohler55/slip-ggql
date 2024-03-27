
all: build

clean:
	rm *.so

lint:
	golangci-lint run

build:
	go build -buildmode=plugin -o ggql.so ./...

test: lint build
	go test -coverprofile=cov.out ./...

.PHONY: all build
