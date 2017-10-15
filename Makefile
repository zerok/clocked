all: clocked

clocked: $(shell find . -name '*.go')
	cd cmd/clocked && go build -o ../../clocked

clean:
	rm -f clocked

install:
	cd cmd/clocked && go install

test:
	go test ./... -v

snapshot:
	goreleaser --skip-validate --skip-publish --snapshot --rm-dist

.PHONY: clean
.PHONY: all
.PHONY: test
.PHONY: snapshot
