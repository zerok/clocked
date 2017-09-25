all: clocked

clocked: $(shell find . -name '*.go')
	cd cmd/clocked && go build -o ../../clocked

clean:
	rm -f clocked

install:
	cd cmd/clocked && go install

test:
	go test ./... -v

.PHONY: clean
.PHONY: all
.PHONY: test
