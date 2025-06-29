#!/bin/bash

#
#  Usage:  $0  <num-workers>  <sleep-time>
#          where:  <num-workers> = number of workers - default 10.
#                  <sleep-time>  = max sleep time in seconds - default 5.
#                                  This is used by the workers to randomly
#                                  sleep between 0 and the max sleep time.
#

#  Constants.
readonly DEFAULT_WORKERS=10
readonly DEFAULT_DELAY_SECS=5
readonly LOGFILE=/tmp/worker.log


function start_sleeper() {
    local stype=${1:-"random"}
    local code=${2:-0}
    local delay=${3:-"${DEFAULT_DELAY_SECS}"}

    local sleeptime=${delay}
    [ "$stype" = "random" ]  &&  sleeptime=$((RANDOM % delay))
 
     nohup sh -c "sleep ${sleeptime} && exit ${code}" < /dev/null &> /dev/null &
     pid=$!
     echo "  - Worker: $$ - started ${stype} sleeper ..." >> "${LOGFILE}"
     echo "    background pid=${pid}, exitcode=${code}" >> "${LOGFILE}"

}  #  End of function  start_sleeper.


function run_workers() {
    local ntimes=${1:-"$DEFAULT_WORKERS"}
    shift

    #  Start 1 fixed and 'n' random sleepers.
    start_sleeper "fixed" 0 "$@"

    #shellcheck disable=SC2034
    for i in $(seq "${ntimes}"); do
        start_sleeper "random" 0 "$@"
    done

    # Test with a bunch of different exit codes.
    for code in 1 2 7 13 21 29 30 31 64 65 66 69 70 71 74 76 77 78 127; do
        start_sleeper "random" "${code}" "$@"
    done

}  #  End of function  run_workers.


#
#  main():  Do the work starting up the appropriate number of workers.
#
run_workers "$@"
