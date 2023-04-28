
all: build

clean:
	rm *.so

build:
	go build -buildmode=plugin -o ggql.so *.go

.PHONY: all build
