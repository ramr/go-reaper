#!/usr/bin/env make

TEST_IMAGE = "reaper/test"
DEBUG_ON_CONFIG_TEST_IMAGE = "reaper/debug-on-config-test"
NON_PID1_CONFIG_TEST_IMAGE = "reaper/non-pid1-config-test"
OOP_INIT_TEST_IMAGE = "reaper/oop-init-config-test"


all:	build

build:	deps lint vet image

clean:
	@echo "  - Removing testpid1 binary ..."
	$(RM) testpid1

	@echo "  - Removing docker images ..."
	docker rmi $(TEST_IMAGE) || :
	docker rmi $(DEBUG_ON_CONFIG_TEST_IMAGE) || :
	docker rmi $(NON_PID1_CONFIG_TEST_IMAGE) || :
	docker rmi $(OOP_INIT_TEST_IMAGE) || :

	@echo "  - Pruning docker environment ..."
	docker system prune -f

	@echo "  - Removing test logs ..."
	$(RM) /tmp/reaper-tests/*.log

test:	tests
tests:	build test-image test-debug-on test-notify test-non-pid1 test-oop-init
	@echo "  - All tests passed."


#
#  Lint and vet targets.
#
lint:
	@echo "  - Linting shell sources ..."
	@(shellcheck bin/*.sh build/*.sh *.sh || :)
	@echo "  - shellcheck passed."

	@echo "  - Linting sources ..."
	gofmt -d -s testpid1.go fixtures/oop-init/testpid1.go
	@echo "  - Linter checks passed."

vet:
	@echo "  - Vetting go sources ..."
	go vet testpid1.go
	go vet fixtures/oop-init/testpid1.go
	@echo "  - go vet checks passed."


#
#  Update dependencies (modules).
#
deps:
	@echo "  - Update dependencies ..."
	go mod tidy

	@echo "  - Download go modules ..."
	go mod download   #  -x

#
#  Build a single image to test the various reaper fixtures configs with.
#
image:
	@echo "  - Rebuilding testpid1.go ..."
	$(RM) testpid1
	go build testpid1.go

	@echo "  - Building docker image $(TEST_IMAGE) ..."
	docker build -t $(TEST_IMAGE) .


#
#  Testing targets.
#
test-image:
	@echo "  - Running reaper image test ..."
	./runtests.sh $(TEST_IMAGE) ""

	@echo "  - Running reaper image test with missing reaper.json ..."
	./runtests.sh $(TEST_IMAGE) /reaper/config/missing-404-reaper.json

test-debug-on:
	@echo "  - Running reaper image debug on test ..."
	./runtests.sh $(TEST_IMAGE) /reaper/config/debug-reaper.json

test-notify:	test-status test-status-close

test-status:
	@echo "  - Running reaper image status test ..."
	./runtests.sh $(TEST_IMAGE) /reaper/config/status-reaper.json

test-status-close:
	@echo "  - Running reaper image status close test ..."
	./runtests.sh $(TEST_IMAGE) /reaper/config/status-close-reaper.json

test-non-pid1:	test-child-sub-reaper
	@echo "  - Running reaper image non pid1 test ..."
	./runtests.sh $(TEST_IMAGE) /reaper/config/non-pid1-reaper.json

test-child-sub-reaper:
	@echo "  - Running reaper image non pid1 child-sub-reaper test ..."
	./runtests.sh $(TEST_IMAGE) /reaper/config/child-sub-reaper.json

#
#  Custom image tests.
#
test-oop-init:
	(build/image.sh $(OOP_INIT_TEST_IMAGE) "fixtures/oop-init";  \
	 ./runtests.sh $(OOP_INIT_TEST_IMAGE) )


.PHONY:	build clean test tests lint vet image
.PHONY:	test-image test-debug-on test-notify test-status test-status-close
.PHONY:	test-non-pid1 test-oop-init
