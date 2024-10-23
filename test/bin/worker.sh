#!/bin/bash

#
#  Usage:  <./worker.sh>  <num-workers>  <sleep-time>
#          where:  <num-workers> = number of workers - default 10.
#                  <sleep-time>  = max sleep time in seconds - default 5.
#                                  This is used by the workers to randomly
#                                  sleep between 0 and the max sleep time.
#

#  Constants.
readonly DEFAULT_WORKERS=10
readonly DEFAULT_DELAY_SECS=5

#  Change log file if you want see the script output.
#    LOGFILE=/tmp/worker.log
LOGFILE=/dev/null


function start_sleeper() {
  local stype=${1:-"random"}
  local exitCode=${2:-0}
  local delay=${3:-"$DEFAULT_DELAY_SECS"}

  local sleeptime=$delay
  [ "$stype" = "random" ]  &&  sleeptime=$((RANDOM % $delay))
 
   nohup sh -c "sleep ${sleeptime} && exit ${exitCode}" < /dev/null &> /dev/null &
   pid=$!
   echo "  - Worker: $$ - started $stype bg sleeper, pid=$pid, exitCode=$exitCode" >> "$LOGFILE"

}  #  End of function  start_sleeper.


function run_workers() {
  local ntimes=${1:-"$DEFAULT_WORKERS"}
  shift

  #  Start 1 fixed and 'n' random sleepers.
  start_sleeper "fixed" 0 "$@"

  for i in $(seq $ntimes); do
     start_sleeper "random" 0 "$@"
  done

  start_sleeper "random" 1 "$@"
  start_sleeper "random" 2 "$@"

}  #  End of function  run_workers.



#
#  main():  Do the work starting up the appropriate number of workers.
#
run_workers "$@"

