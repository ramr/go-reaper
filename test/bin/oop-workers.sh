#!/bin/bash

SCRIPT_DIR=$(dirname "${BASH_SOURCE[0]}")


# Run batched, so that we create a new process group id/session id.
set -m

echo "grepping root ..."
cat /etc/passwd | grep root | sort | grep root

echo "starting worker ..."
nohup bash -c "$SCRIPT_DIR/worker.sh $@" < /dev/null &> /dev/null &
pid=$!
echo "  - Started background worker - pid=$pid"
set +m
