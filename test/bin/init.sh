#!/bin/bash


#
# Check options to figure out if we need to run the reaper as a child
# process (!pid-1) or as the first process (as pid 1 replacing this script).
#
function _start_reaper() {
    local cfg=${1:-""}
    local forked=0

    if [ -f "${cfg}" ]; then
        echo "  - Checking if we need to run reaper as a child process ..."
        if grep -Ee '"DisablePid1Check":\s*true' "${cfg}" ; then
            forked=1
        fi
    fi

    if [ "${forked}" -eq 1 ]; then
        echo "  - Running reaper as child process ..."

        # Run testpid1 with a new parent and process group id/session id.
        bash <<EOF
            set -m
            echo "  - New session pid = \$\$"
            echo "  - Starting testpid1 ..."
            /reaper/testpid1 "$@"
EOF

        exit $?
    fi


    echo "  - Replacing init.sh with reaper as pid 1 ..."
    exec /reaper/testpid1 "$1" "$@"

}  #  End of function  _start_reaper.


#
#  main():
#
_start_reaper "$@"
