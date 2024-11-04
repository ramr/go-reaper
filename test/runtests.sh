#!/bin/bash

set -euo pipefail


readonly MAX_SLEEP_TIME=$((5 + 2))
readonly IMAGE="reaper/test"

logfile="/tmp/reaper-tests/test.log"


#
#  Return list of sleeper processes.
#
function _get_sleepers() {
    #shellcheck disable=SC2009
    ps --forest -o pid,ppid,time,cmd -g "${pid1}" |  \
        grep sleep | grep -v grep

}  #  End of function  _get_sleepers.


#
#  Check for orphaned processes.
#
function _check_orphans() {
    local pid1=$1

    sleep "${MAX_SLEEP_TIME}"
    local orphans=""
    orphans=$(_get_sleepers "${pid1}")

    if [ -n "${orphans}" ]; then
        echo ""
        echo "FAIL: Got orphan processes attached to pid ${pid1}"
        echo "============================================================="
        echo "${orphans}"
        echo "============================================================="
        echo "      No sleep processes expected."
        return 1
    fi

    return 0

}  #  End of function  _check_orphans.


#
#  Terminate docker container.
#
function _terminate_container() {
    local logdir=""
    logdir=$(dirname "${logfile}")

    mkdir -p "${logdir}"
    docker logs "$1" > "${logfile}"

    #  append worker logs to the logfile.
    (echo ""; echo "";                                          \
        echo "----------------------------------------------";  \
        echo "  - Worker logs:";                                \
        docker exec "$1" cat /tmp/worker.log;                   \
        echo "----------------------------------------------";  \
    ) >> "${logfile}"

    echo "  - Container logs saved to ${logfile}"

    echo "  - Terminated container $(docker rm -f "$1")"

}  #  End of function  _terminate_container.


#
#  If we have status notifications turned on (and are not randomly closing
#  the status channel), then check whether we got the various different
#  exit codes from the reaped descendants.
#
function _check_status_exit_codes() {
    local name=${1:-""}

    if [ "z${name}" != "zstatus-reaper" ] &&  \
       [ "z${name}" != "zchild-sub-reaper" ]; then
        #  Nothing to do. Test doesn't have any status notifications, so
        #  the log won't contain any reaped descendant exit codes.
        return 0
    fi

    #  Check that we have expected status lines.
    for code in 1 2 7 13 21 29 30 31 64 65 66 69 70 71 74 76 77 78 127; do
        local pid=""
        for pid in $(grep -Ee "exitcode=${code}\$" "${logfile}" |  \
                         awk -F '=' '{print $2}' |                 \
                         awk -F ',' '{print $1}' | tr '\r\n' ' '); do
            #echo "  - Descendant pid ${pid} exited with code ${code}"

            #  On a slower vm (circa 2014), might not always get the status
            #  reported - it could fill up the channel notifications, so ...
            if grep "status of pid ${pid}" "${logfile}" > /dev/null; then
                local reported=""
                reported=$(grep "status of pid ${pid}" "${logfile}" |   \
                               sed 's/.*status of pid.*exit code //g' |  \
                               tr -d '\r')
                if [ "${reported}" -ne "${code}" ]; then
                    echo "ERROR: Child pid ${pid} exited with code ${code},"
                    echo "       but reported code is ${reported}"
                    exit 65
                fi
            fi
        done
    done

    return 0

}  #  End of function  _check_status_exit_codes.


#
#  Run reaper tests.
#
function _run_tests() {
    local image=${1:-"${IMAGE}"}
    shift

    local name="test"
    name=$(echo "${image}" | awk -F '/' '{print $NF}')

    local config=${1:-""}
    if [ -n "${config}" ]; then
        config=$(basename "${config}")
        name="${config%.*}"
    fi

    logfile="/tmp/reaper-tests/${name}.log"

    echo "  - Removing any existing log file ${logfile} ... "
    rm -f "${logfile}"

    echo "  - Starting docker container running image ${image} with"
    echo "    command line arguments: $*"
    local elcid=""
    elcid=$(docker run -dit "${image}" "$@")

    echo "  - Docker container name=${elcid}"
    local pid1=""
    pid1=$(docker inspect --format '{{.State.Pid}}' "${elcid}")

    sleep 1.42

    echo "  - Docker container pid=${pid1}"
    echo "  - PID ${pid1} has $(_get_sleepers "${pid1}" | wc -l) sleepers."
    echo "  - Sleeping for ${MAX_SLEEP_TIME} seconds ..."
    sleep "${MAX_SLEEP_TIME}"

    echo "  - Checking for orphans attached to pid1=${pid1} ..."
    if ! _check_orphans "${pid1}"; then
        #  Got an error, cleanup and exit with error code.
        _terminate_container "${elcid}"
        echo ""
        echo "FAIL: All tests failed - (1/1)"
        exit 65
    fi
 

    local cname=""
    cname=$(echo "${elcid}" | cut -c 1-12)
    echo "  - Sending SIGUSR1 to ${cname} (pid ${pid1}) to start more workers ..."
    docker kill -s USR1 "${elcid}"

    sleep 1
    echo "  - PID ${pid1} has $(_get_sleepers "${pid1}" | wc -l) sleepers."

    echo "  - Sleeping once again for ${MAX_SLEEP_TIME} seconds ..."
    sleep "${MAX_SLEEP_TIME}"

    echo "  - Running processes under ${pid1}:"
    pstree "${pid1}"
    ps --forest -o pid,ppid,time,cmd -g "${pid1}" || :

    echo "  - Checking for orphans attached to pid1=${pid1} ..."
    if ! _check_orphans "${pid1}"; then
        #  Got an error, cleanup and exit with error code.
        _terminate_container "${elcid}"
        echo ""
        echo "FAIL: Some tests failed - (1/2)"
        exit 65
    fi

    echo "  - Running processes under ${pid1}:"
    pstree "${pid1}"
    ps --forest -o pid,ppid,time,cmd -g "${pid1}" || :

    #  Do the cleanup.
    _terminate_container "${elcid}"

    #  If we have the status, check the different exit codes.
    _check_status_exit_codes  "${name}"

    echo ""
    echo "OK: All tests passed - (2/2)"

} #  End of function  _run_tests.


#
#  main():
#
_run_tests "$@"
