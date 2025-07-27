#!/bin/bash

set -euo pipefail

SCRIPT_DIR=$(cd -P "$(dirname "${BASH_SOURCE[0]}")" && pwd)
readonly SCRIPT_DIR

readonly MAX_SLEEP_TIME=$((5 + 2))
readonly IMAGE="reaper/test"

readonly SCENARIO_LOG="/tmp/reaper-tests/test-scenario.log"
readonly WORKER_LOG="/tmp/reaper-tests/worker.log"

logfile="/tmp/reaper-tests/test.log"


#
#  Run test locally [on host].
#
function _run_local_test() {
    local testpid=-1
    local cfg="./fixtures/config/local-test.json"

    echo "  - Starting local test process in the background ..."
    echo "    Log file = ${SCENARIO_LOG}"
    "${SCRIPT_DIR}/testpid1"  "${cfg}" > "${SCENARIO_LOG}" 2>&1  &
    testpid=$!

    echo "  - Local test process pid = ${testpid}"

    echo "  - Snoozing ..."
    sleep "${MAX_SLEEP_TIME}"

    echo "  - Sending SIGTERM to the child of the test process ..."

    #  Lil' hacky but it does the job.
    "${SCRIPT_DIR}/bin/send-child-signal.sh" TERM &

    local exitcode=0

    echo "  - Waiting for local test process ${testpid} to exit ..."
    wait "${testpid}"
    exitcode=$?

    echo "  - Local test process ${testpid} exit code = ${exitcode}"
    return ${exitcode}

}  #  End of function  _run_local_test.


#
#  Return list of sleeper processes.
#
function _get_sleepers() {
    docker exec "$1" pgrep sleep

}  #  End of function  _get_sleepers.


#
#  Check for orphaned processes.
#
function _check_orphans() {
    local elcid=$1

    local cname=""
    cname=$(echo "${elcid}" | cut -c 1-12)

    sleep "${MAX_SLEEP_TIME}"
    local orphans=""
    orphans=$(_get_sleepers "${elcid}")

    if [ -n "${orphans}" ]; then
        echo ""
        echo "FAIL: Got orphan processes attached to reaper in ${cname}"
        echo "============================================================="
        echo "${orphans}"
        echo "============================================================="
        echo "      No sleep processes expected."
        return 1
    fi

    return 0

}  #  End of function  _check_orphans.


#
#  Returns 0 if ErrorExit is set in the associated config.
#
function _errorExitEnabled() {
    local config=${1:-""}

    if [ -z "${config}" ]; then
        return 1
    fi

    local cfgjson="${SCRIPT_DIR}/fixtures/config/${config}"
    if [ -f "${cfgjson}" ]; then
        if grep -Ee '"ErrorExit":\s*true' "${cfgjson}" ; then
            #  The death of this process is pre-ordained!
            return 0
        fi
    fi

    return 1

}  #  End of function  _errorExitEnabled.


#
#  Save container worker logs.
#
function _save_container_worker_logs() {
    local logdir=""
    logdir=$(dirname "${WORKER_LOG}")

    mkdir -p "${logdir}"

    local running=""
    running=$(docker inspect "$1" -f '{{ .State.Running }}')
    if [ "z${running}" == "ztrue" ]; then
        (echo ""; echo "";                                          \
            echo "----------------------------------------------";  \
            echo "  - Worker logs:";                                \
            docker exec "$1" cat /tmp/worker.log;                   \
            echo "----------------------------------------------";  \
        ) > "${WORKER_LOG}"
    fi

    return 0

}  #  End of function  _save_container_worker_logs.


#
#  Terminate docker container.
#
function _terminate_container() {
    local logdir=""
    logdir=$(dirname "${SCENARIO_LOG}")

    mkdir -p "${logdir}"
    docker logs "$1" > "${SCENARIO_LOG}"

    _save_container_worker_logs "$1"

    if [ -f "${WORKER_LOG}" ]; then
        #  append worker logs to the SCENARIO_LOG.
        cat "${WORKER_LOG}" >> "${SCENARIO_LOG}"
    fi

    echo "  - Container logs saved to ${SCENARIO_LOG}"

    #echo "  - container not terminated - for debugging!"
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
        for pid in $(grep -Ee "exitcode=${code}\$" "${SCENARIO_LOG}" |  \
                         awk -F '=' '{print $2}' |                 \
                         awk -F ',' '{print $1}' | tr '\r\n' ' '); do
            #echo "  - Descendant pid ${pid} exited with code ${code}"

            #  On a slower vm (circa 2014), might not always get the status
            #  reported - it could fill up the channel notifications, so ...
            if grep "status of pid ${pid}" "${SCENARIO_LOG}" > /dev/null; then
                local reported=""
                reported=$(grep "status of pid ${pid}" "${SCENARIO_LOG}" | \
                               sed 's/.*status of pid.*exit code //g' |   \
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
#  Run a reaper test scenario.
#
function _run_test_scenario() {
    local image=${1:-"${IMAGE}"}
    shift

    local name="test"
    name=$(echo "${image}" | awk -F '/' '{print $NF}')

    local config=${1:-""}
    if [ -n "${config}" ]; then
        config=$(basename "${config}")
        name="${config%.*}"
    fi

    echo "  - Logging full test logs to ${logfile} ..."
    echo "  - Removing any existing ${SCENARIO_LOG} ${WORKER_LOG} ... "
    rm -f "${SCENARIO_LOG}" "${WORKER_LOG}"

    local imageid=""
    if [ "osx-$(uname -s)" == "osx-Darwin" ]; then
        if [ ":${name}" != ":on-host-test" ]; then
            echo "  - Skipping ${name} test on osx ..."  |  \
                tee -a "${SCENARIO_LOG}"
            return 0
        fi
    else
        imageid=$(docker image ls -q "${image}" || :)
    fi

    if [ -z "${imageid}" ]; then
        echo "  - Running local [on-host] test ..."

        if _run_local_test ; then
            echo ""
            echo "OK: All tests passed - (1/1)"
            exit 0
        fi

        echo ""
        echo "FAIL: Local tests failed - (1/1)"
        exit 65
    fi

    echo "  - Starting docker container running image ${image} with"
    echo "    command line arguments: $*"
    local elcid=""
    elcid=$(docker run -dit "${image}" "$@")

    local cname=""
    cname=$(echo "${elcid}" | cut -c 1-12)

    echo "  - Docker container name=${cname}"
    local pid1=""
    pid1=$(docker inspect --format '{{.State.Pid}}' "${elcid}")

    sleep 1.42

    echo "  - Docker container ${cname}, host process pid=${pid1}"
    echo "  - ${cname} has $(_get_sleepers "${elcid}" | wc -l) sleepers."

    if [ "${pid1}" != "0" ]; then
        echo "  - Sleeping for ${MAX_SLEEP_TIME} seconds ..."
        sleep "${MAX_SLEEP_TIME}"
    fi

    #  Check if we are expecting error exits.
    local errorExit="no"
    if _errorExitEnabled "${config}" ; then
        errorExit="yes"

        if [ "${pid1}" == "0" ] || ! ps -p "${pid1}" > /dev/null ; then
            #  Already dead, then do the cleanup.
            _terminate_container "${elcid}"

            echo ""
            echo "OK: All tests passed - (1/1)"
            return
        fi
    fi

    echo "  - Checking for orphans attached to container ${cname} ..."
    if ! _check_orphans "${elcid}"; then
        #  Got an error, cleanup and exit with error code.
        _terminate_container "${elcid}"
        echo ""
        echo "FAIL: All tests failed - (1/1)"
        exit 65
    fi
 

    echo "  - Running add more workers in ${cname} ..."
    docker exec "${elcid}" /reaper/bin/send-child-signal.sh USR1

    sleep 1.42
    echo "  - ${cname} has $(_get_sleepers "${elcid}" | wc -l) sleepers."

    echo "  - Sleeping once again for ${MAX_SLEEP_TIME} seconds ..."
    sleep "${MAX_SLEEP_TIME}"

    echo "  - Running processes in container ${cname}:"
    pstree "${pid1}"
    docker exec "${elcid}" ps --forest -o pid,ppid,time,cmd -e || :

    echo "  - Checking for orphans attached to container ${cname} ..."
    if ! _check_orphans "${elcid}"; then
        #  Got an error, cleanup and exit with error code.
        _terminate_container "${elcid}"
        echo ""
        echo "FAIL: Some tests failed - (1/2)"
        exit 65
    fi

    echo "  - Running processes in container ${cname}:"
    pstree "${pid1}"
    docker exec "${elcid}" ps --forest -o pid,ppid,time,cmd -e || :

    if [ "z${errorExit}" == "zyes" ]; then
        _save_container_worker_logs "${elcid}"

        echo "  - Send child signal SIGTERM"
        docker exec "${elcid}" /reaper/bin/send-child-signal.sh TERM
        sleep 1.42
    fi

    #  Do the cleanup.
    _terminate_container "${elcid}"

    #  If we have the status, check the different exit codes.
    _check_status_exit_codes  "${name}"

    echo ""
    echo "OK: All tests passed - (2/2)"

} #  End of function  _run_test_scenario.


#
#  Run reaper tests.
#
function _run_tests() {
    local image=${1:-"${IMAGE}"}
    local name="test"
    name=$(echo "${image}" | awk -F '/' '{print $NF}')

    local config=${2:-""}
    if [ -n "${config}" ]; then
        config=$(basename "${config}")
        name="${config%.*}"
    fi

    logfile="/tmp/reaper-tests/${name}.log"
    mkdir -p "$(dirname "${logfile}")"

    _run_test_scenario  "$@"  | tee "${logfile}"
    cat "${SCENARIO_LOG}" >> "${logfile}"

} #  End of function  _run_tests.


#
#  main():
#
_run_tests "$@"
