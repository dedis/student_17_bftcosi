package protocol

import (
	"gopkg.in/dedis/onet.v1"
	"errors"
	"fmt"
)

// GenTree will create a tree of n servers with a localRouter, and returns the
// list of servers and the associated roster / tree.
// NOTE: register being not implementable with the current API could hurt the scalability tests
func GenTree(roster *onet.Roster, nNodes, nShards int) (error, *onet.Tree) {

	//parameter verification
	if roster == nil {
		return errors.New("the roster is nil"), nil
	}
	if nNodes < 1 {
		return fmt.Errorf("the number of nodes in a tree " +
			"cannot be less than one, but is %d", nNodes), nil
	}
	if len(roster.List) < nNodes {
		return fmt.Errorf("the tree should have %d nodes, but there is only %d servers in the roster",
			nNodes, len(roster.List)), nil
	}
	if nShards < 1 {
		return fmt.Errorf("the number of shards in a tree " +
			"cannot be less than one, but is %d", nShards), nil
	}

	if nNodes <= nShards {
		nShards = nNodes -1
	}

	//generate first level of the tree
	nTopLevelNodes := nShards +1
	rootNode := onet.NewTreeNode(0, roster.List[0])
	for i := 1 ; i< nTopLevelNodes; i++ {
		node := onet.NewTreeNode(i, roster.List[i])
		node.Parent = rootNode
		rootNode.Children = append(rootNode.Children, node)
	}


	//generate each shard
	if nTopLevelNodes != nNodes {

		nodesPerShard := (nNodes - 1) / nShards
		surplusNodes := (nNodes - 1) % nShards

		start := nTopLevelNodes
		for i, n := range rootNode.Children {

			end := start + (nodesPerShard -1)
			if i< surplusNodes { //to handle surplus nodes
				end++
			}

			for j := start ; j < end ; j++ {
				node := onet.NewTreeNode(j, roster.List[j])
				node.Parent = n
				n.Children = append(n.Children, node)
			}
			start = end
		}
	}

	tree := onet.NewTree(roster, rootNode)

	//l.Trees[tree.ID] = tree
	//if registerOLD {
	//	servers[0].overlay.RegisterRoster(list)
	//	servers[0].overlay.RegisterTree(tree)
	//}

	return nil, tree
}
