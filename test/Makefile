#!/usr/bin/env make

TEST_IMAGE = "reaper/test"
CONFIG_TEST_IMAGE = "reaper/config-test"

all:	tests

test:	tests

tests:
	(go build testpid1.go;                       \
	  cp docker-files/no-config/Dockerfile .;    \
	  docker build -t $(TEST_IMAGE) .;           \
	  ./runtests.sh;                             \
	  cp docker-files/json-config/Dockerfile .;  \
	  docker build -t $(CONFIG_TEST_IMAGE) .;    \
	  ./runtests.sh "$(CONFIG_TEST_IMAGE)" )
