#!/usr/bin/env make

all:	build

build:	lint vet

clean:
	(cd test; $(MAKE) clean)

test:	tests
tests:	lint vet integration-tests


#
#  Lint and vet targets.
#
lint:
	(cd test && $(MAKE) lint)

	@echo  "  - Linting sources ..."
	gofmt -d -s reaper.go
	@echo  "  - Linter checks passed."

vet:
	(cd test && $(MAKE) vet)

	@echo  "  - Vetting go sources ..."
	go vet ./...
	@echo  "  - go vet checks passed."


#
#  Test targets.
#
integration-test:	integration-tests
integration-tests:
	(cd test && $(MAKE))


.PHONY:	build clean test tests lint vet integration-test integration-tests
