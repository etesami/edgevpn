// Copyright © 2021-2022 Ettore Di Giacinto <mudler@mocaccino.org>
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

package vpn

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/ipfs/go-log/v2"
	"edgevpn/pkg/crypto"
	"edgevpn/pkg/node"
	"edgevpn/pkg/protocol"
	"edgevpn/pkg/services"
	"edgevpn/pkg/types"
	"edgevpn/pkg/utils"

	"edgevpn/pkg/blockchain"
)

func checkDHCPLease(c node.Config, leasedir string) string {
	// retrieve lease if present

	leaseFileName := crypto.MD5(fmt.Sprintf("%s-ek", c.ExchangeKey))
	leaseFile := filepath.Join(leasedir, leaseFileName)
	if _, err := os.Stat(leaseFile); err == nil {
		b, _ := ioutil.ReadFile(leaseFile)
		return string(b)
	}
	return ""
}
func contains(slice []string, elem string) bool {
	for _, s := range slice {
		if elem == s {
			return true
		}
	}
	return false
}

// DHCPNetworkService returns a DHCP network service
func DHCPNetworkService(ip chan string, l log.StandardLogger, maxTime time.Duration, leasedir string, address string) node.NetworkService {
	return func(ctx context.Context, c node.Config, n *node.Node, b *blockchain.Ledger) error {
		l.Debug("Starting DHCP Network Service")
		os.MkdirAll(leasedir, 0600)

		// retrieve lease if present
		var wantedIP = checkDHCPLease(c, leasedir)

		//  whoever wants a new IP:
		//  1. Get available nodes. Filter from Machine those that do not have an IP.
		//  2. Get the leader among them. If we are not, we wait
		//  3. If we are the leader, pick an IP and start the VPN with that IP
		for wantedIP == "" {
			time.Sleep(5 * time.Second)

			// This network service is blocking and calls in before VPN, hence it needs to registered before VPN
			nodes := services.AvailableNodes(b, maxTime)

			currentIPs := map[string]string{}
			ips := []string{}

			for _, t := range b.LastBlock().Storage[protocol.MachinesLedgerKey] {
				var m types.Machine
				t.Unmarshal(&m)
				currentIPs[m.PeerID] = m.Address

				l.Debugf("%s uses %s", m.PeerID, m.Address)
				ips = append(ips, m.Address)
			}

			nodesWithNoIP := []string{}
			for _, nn := range nodes {
				if _, exists := currentIPs[nn]; !exists {
					nodesWithNoIP = append(nodesWithNoIP, nn)
				}
			}

			if len(nodes) <= 1 {
				l.Debug("not enough nodes to determine an IP, sleeping")
				continue
			}

			if len(nodesWithNoIP) == 0 {
				l.Debug("not enough nodes waiting for IP being announced, sleeping")
				continue
			}

			shouldBeLeader := utils.Leader(nodesWithNoIP)

			var lead string
			v, exists := b.GetKey("dhcp", "leader")
			if exists {
				v.Unmarshal(&lead)
			}

			if shouldBeLeader != n.Host().ID().String() && lead != n.Host().ID().String() {
				c.Logger.Infof("<%s> not a leader, leader is '%s', sleeping", n.Host().ID().String(), shouldBeLeader)
				continue
			}

			if shouldBeLeader == n.Host().ID().String() && (lead == "" || !contains(nodesWithNoIP, lead)) {
				b.Persist(ctx, 5*time.Second, 15*time.Second, "dhcp", "leader", n.Host().ID().String())
				c.Logger.Info("Announcing ourselves as leader, backing off")
				continue
			}

			if lead != n.Host().ID().String() {
				c.Logger.Info("Backing off, as we are not currently flagged as leader")
				time.Sleep(5 * time.Second)
				continue
			}

			l.Debug("Nodes with no ip", nodesWithNoIP)
			// We are lead
			l.Debug("picking up between", ips)

			wantedIP = utils.NextIP(address, ips)
		}

		// Save lease to disk
		leaseFileName := crypto.MD5(fmt.Sprintf("%s-ek", c.ExchangeKey))
		leaseFile := filepath.Join(leasedir, leaseFileName)
		l.Debugf("Writing lease to '%s'", leaseFile)
		if err := ioutil.WriteFile(leaseFile, []byte(wantedIP), 0600); err != nil {
			l.Warn(err)
		}

		// propagate ip to channel that is read while starting vpn
		ip <- wantedIP

		// Gate connections from VPN
		return n.BlockSubnet(fmt.Sprintf("%s/24", wantedIP))
	}
}

// DHCP returns a DHCP network service. It requires the Alive Service in order to determine available nodes.
// Nodes available are used to determine which needs an IP and when maxTime expires nodes are marked as offline and
// not considered.
func DHCP(l log.StandardLogger, maxTime time.Duration, leasedir string, address string) ([]node.Option, []Option) {
	ip := make(chan string, 1)
	return []node.Option{
			func(cfg *node.Config) error {
				// retrieve lease if present. consumed by conngater when starting the node
				lease := checkDHCPLease(*cfg, leasedir)
				if lease != "" {
					cfg.InterfaceAddress = fmt.Sprintf("%s/24", lease)
				}
				return nil
			},
			node.WithNetworkService(DHCPNetworkService(ip, l, maxTime, leasedir, address)),
		}, []Option{
			func(cfg *Config) error {
				// read back IP when starting vpn
				cfg.InterfaceAddress = fmt.Sprintf("%s/24", <-ip)
				close(ip)
				l.Debug("IP Received", cfg.InterfaceAddress)
				return nil
			},
		}
}
