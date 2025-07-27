//go:build darwin
// +build darwin

package reaper

/*  Note:  This is darwin specific version [just returns an error]. */

import (
	"fmt"
)

// Enable child subreaper.
func EnableChildSubReaper() error {
	return fmt.Errorf("child subreaper not supported on darwin")

} /*  End of [exported] function  EnableChildSubReaper.  */
