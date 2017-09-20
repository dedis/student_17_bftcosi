package protocol

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

	//generate servers
	servers := l.GenServers(n_nodes)
	roster := l.GenRosterFromHost(servers...)

	//generate first level of the tree
	n_top_level_nodes := n_shards+1
	root_node := onet.NewTreeNode(0, roster.List[0])
	for i := range servers[:n_top_level_nodes] {
		node := onet.NewTreeNode(i, roster.List[i])
		if i > 0 {
			node.Parent = root_node
			root_node.Children = append(root_node.Children, node)
		}
	}



	if n_top_level_nodes != n_nodes {

		nodes_per_shard := (n_nodes - 1) / n_shards

		//generate each shard
		for i, n := range root_node.Children {
			start := i*(nodes_per_shard-1) + n_top_level_nodes
			end := start + (nodes_per_shard-1)

			for j := start ; j < end ; j++ {
				node := onet.NewTreeNode(j, roster.List[j])
				node.Parent = n
				root_node.Children = append(n.Children, node)
			}
		}
	}

	tree := onet.NewTree(roster, root_node)

	l.Trees[tree.ID] = tree
	if register {
		//servers[0].overlay.RegisterRoster(list)
		//servers[0].overlay.RegisterTree(tree)
	}

	return servers, roster, tree
}
