package protocol //TODO: protocol_test ?

import (
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/network"
)

// GenTree will create a tree of n servers with a localRouter, and returns the
// list of servers and the associated roster / tree.
//TODO: make register work
func GenTree(l *onet.LocalTest, n_nodes, n_shards int, register bool) ([]*onet.Server, *onet.Roster, *onet.Tree) {

	if n_nodes < n_shards {
		n_shards = n_nodes-1
	}

	//generate top-level of the tree
	servers := l.GenServers(n_nodes)
	n_top_level_nodes := n_shards+1
	roster := l.GenRosterFromHost(servers[:n_top_level_nodes]...)
	tree := roster.GenerateNaryTree(n_shards)
	l.Trees[tree.ID] = tree
	if register {
		//servers[0].overlay.RegisterRoster(list)
		//servers[0].overlay.RegisterTree(tree)
	}

	if n_top_level_nodes != n_nodes {

		nodes_per_shard := (n_nodes - 1) / n_shards

		//generate each shard
		for i, s := range servers[1:n_top_level_nodes] {
			start := i*(nodes_per_shard-1) + n_top_level_nodes
			end := start + (nodes_per_shard-1)
			shards_servers := append(servers[start:end], s)

			shard_roster := l.GenRosterFromHost(shards_servers...)
			shard_roster.GenerateNaryTreeWithRoot(nodes_per_shard, s.ServerIdentity)
			if register {
				//servers[0].overlay.RegisterRoster(shard_list)
				//servers[0].overlay.RegisterTree(shard_tree)
			}
		}

		var server_identities []*network.ServerIdentity
		for _, server := range servers {
			server_identities = append(server_identities, server.ServerIdentity)
		}

		roster = onet.NewRoster(server_identities)
		tree.Roster = roster
	}

	return servers, roster, tree
}
