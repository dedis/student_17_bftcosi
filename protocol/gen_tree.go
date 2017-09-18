package protocol //TODO: protocol_test ?

import (
	"gopkg.in/dedis/onet.v1"
)

// GenTree will create a tree of n servers with a localRouter, and returns the
// list of servers and the associated roster / tree.
//TODO: make register work
func GenTree(l *onet.LocalTest, n int, register bool) ([]*onet.Server, *onet.Roster, *onet.Tree) {
	servers := l.GenServers(n)

	list := l.GenRosterFromHost(servers...)
	tree := list.GenerateNaryTree(2)
	l.Trees[tree.ID] = tree
	if register {
		//servers[0].overlay.RegisterRoster(list)
		//servers[0].overlay.RegisterTree(tree)
	}
	return servers, list, tree

}



