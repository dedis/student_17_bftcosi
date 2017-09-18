package protocol //TODO: protocol_test ?

import (
	"gopkg.in/dedis/onet.v1"
)

// GenTree will create a tree of n servers with a localRouter, and returns the
// list of servers and the associated roster / tree.
//TODO: make register work
func GenTree(l *onet.LocalTest, n_nodes, n_shards int, register bool) ([]*onet.Server, *onet.Roster, *onet.Tree) {

	if n_nodes < n_shards {
		n_shards = n_nodes-1
	}

	//generate top-level of the tree
	servers := l.GenServers(n_shards+1)
	list := l.GenRosterFromHost(servers...)
	tree := list.GenerateNaryTree(n_shards)
	l.Trees[tree.ID] = tree
	if register {
		//servers[0].overlay.RegisterRoster(list)
		//servers[0].overlay.RegisterTree(tree)
	}

	return servers, list, tree
}
