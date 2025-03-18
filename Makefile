.PHONY: build clean test run lint

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOTEST = $(GOCMD) test
BINARY_NAME = cortex

all: build

build:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd/cortexd

test:
	$(GOTEST) -v ./...

run:
	./$(BINARY_NAME) serve

clean:
	rm -f $(BINARY_NAME)

lint:
	gofmt -w -s .
	go vet ./...

tidy:
	go mod tidy 