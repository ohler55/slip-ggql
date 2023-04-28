
all: build

clean:
	rm *.so

build:
	go build -buildmode=plugin -o slip-ggql.so *.go

.PHONY: all build
