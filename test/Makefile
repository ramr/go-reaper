#!/usr/bin/env make

TEST_IMAGE = "reaper/test"
DEBUG_ON_CONFIG_TEST_IMAGE = "reaper/debug-on-config-test"
NON_PID1_CONFIG_TEST_IMAGE = "reaper/non-pid1-config-test"

all:	tests

clean:
	docker rmi $(TEST_IMAGE) || :
	docker rmi $(DEBUG_ON_CONFIG_TEST_IMAGE) || :
	docker rmi $(NON_PID1_CONFIG_TEST_IMAGE) || :

test:	tests

tests:	image-test debug-on-test non-pid1-test

image-test:
	(build/image.sh "$(TEST_IMAGE)" "fixtures/no-config";  \
	 ./runtests.sh "$(TEST_IMAGE)" )

debug-on-test:
	(build/image.sh "$(DEBUG_ON_CONFIG_TEST_IMAGE)"   \
	                "fixtures/json-config/debug-on";  \
	 ./runtests.sh "$(DEBUG_ON_CONFIG_TEST_IMAGE)" )

non-pid1-test:
	(build/image.sh "$(NON_PID1_CONFIG_TEST_IMAGE)"   \
	                "fixtures/json-config/non-pid1";  \
	 ./runtests.sh "$(NON_PID1_CONFIG_TEST_IMAGE)" )
