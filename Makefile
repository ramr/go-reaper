#!/usr/bin/env make

all:	tests

test:	tests

tests:
	(cd test; make)

