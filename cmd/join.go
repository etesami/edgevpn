// Copyright © 2021 Ettore Di Giacinto <mudler@mocaccino.org>
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"context"

	"github.com/mudler/edgevpn/pkg/edgevpn"
	"github.com/urfave/cli"
)

func Join() cli.Command {
	return cli.Command{
		Name:  "join",
		Usage: "Join the network without activating any interface",
		Description: `Connect over the p2p network without establishing a VPN.
Useful for setting up relays or hop nodes to improve the network connectivity.`,
		UsageText: "edgevpn join",
		Flags:     CommonFlags,
		Action: func(c *cli.Context) error {
			e := edgevpn.New(cliToOpts(c)...)

			displayStart(e)

			// Join the node to the network, using our ledger
			if err := e.Join(context.Background()); err != nil {
				return err
			}

			for {
			}
		},
	}
}
