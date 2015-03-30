#!/bin/bash


readonly MAX_SLEEP_TIME=$((5 + 2))
readonly IMAGE="reaper/test"


function check_orphans() {
  local pid1=$1

  sleep $MAX_SLEEP_TIME
  local orphans=$(ps -ef -p $pid1 | grep sleep | grep -v grep)
  if [ -n "$orphans" ]; then
    echo ""
    echo "FAIL: Got orphan processes attached to pid $pid1"
    echo "================================================================="
    echo "$orphans"
    echo "================================================================="
    echo "      No sleep processes expected."
    return 1
  fi

  return 0

}  #  End of function  check_orphans.


function terminate_container() {
    echo "  - Terminated container $(docker rm -f "$1")"

}  #  End of function  terminate_container.


function run_tests() {
  echo "  - Starting docker container running image $IMAGE ..."
  local elcid=$(docker run -dit $IMAGE)
  
  echo "  - Docker container name=$elcid"
  local pid1=$(docker inspect --format '{{.State.Pid}}' $elcid)

  echo "  - Docker container pid=$pid1"
  echo "  - Sleeping for $MAX_SLEEP_TIME seconds ..."
  sleep $MAX_SLEEP_TIME

  echo "  - Checking for orphans attached to pid1=$pid1 ..."
  if ! check_orphans "$pid1"; then
    #  Got an error, cleanup and exit with error code.
    terminate_container "$elcid"
    echo ""
    echo "FAIL: All tests failed - (1/1)"
    exit -1
  fi
  

  echo "  - Sending SIGUSR1 to pid1=$pid1 to start more workers ..."
  kill -USR1 "$pid1"

  echo "  - Sleeping once again for $MAX_SLEEP_TIME seconds ..."
  sleep $MAX_SLEEP_TIME

  echo "  - Checking for orphans attached to pid1=$pid1 ..."
  if ! check_orphans "$pid1"; then
    #  Got an error, cleanup and exit with error code.
    terminate_container "$elcid"
    echo ""
    echo "FAIL: Some tests failed - (1/2)"
    exit -1
  fi

  #  Do the cleanup.
  terminate_container "$elcid"

  echo ""
  echo "OK: All tests passed - (2/2)"

} #  End of function  run_tests.


#
#  main():
#
run_tests "$@"

