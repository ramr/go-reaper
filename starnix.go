//go:build linux
// +build linux

package reaper

/*  Note:  This is a *nix only implementation.  */

import (
	"golang.org/x/sys/unix"
)

// Enable child subreaper.
func EnableChildSubReaper() error {
	return unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0)

} /*  End of [exported] function  EnableChildSubReaper.  */
