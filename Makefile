#!/usr/bin/env make

all:

clean:
	(cd test; make clean)

test:	tests
tests:	lint integration-tests

integration-tests:
	(cd test; make)

lint:
	gofmt -d -s reaper.go
	gofmt -d -s ./test/fixtures/oop-init/testpid1.go ./test/testpid1.go
