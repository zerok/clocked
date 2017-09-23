all: clocked

clocked: $(shell find . -name '*.go')
	cd cmd/clocked && go build -o ../../clocked

clean:
	rm -f clocked

.PHONY: clean
.PHONY: all
