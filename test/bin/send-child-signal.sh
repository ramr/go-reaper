#!/bin/bash


#
#  main():
#
zpid=$(pgrep -a testpid1 | awk '{print $1}' | tail -n 1)
[ -n "${zpid}" ] || zpid=1

sig=${1:-"USR1"}
echo "  - Sending pid ${zpid} signal SIG${sig} ..."
kill -s "${sig}" "${zpid}"
