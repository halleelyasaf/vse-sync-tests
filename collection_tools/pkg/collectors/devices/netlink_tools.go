// SPDX-License-Identifier: GPL-2.0-or-later

package devices

import (
	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/clients"
)

// NetlinkToolsAvailable reports whether DPLL ynl netlink tools exist in this exec context.
func NetlinkToolsAvailable(ctx clients.ExecContext) bool {
	if ctx == nil {
		return false
	}

	checks := [][]string{
		{"test", "-x", dpllYNLCLIPath},
		{"test", "-f", dpllYNLSpecPath},
	}
	for _, cmd := range checks {
		_, _, err := ctx.ExecCommand(cmd)
		if err != nil {
			return false
		}
	}

	return true
}
