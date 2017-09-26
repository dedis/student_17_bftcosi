package protocol_test

/*
The test-file should at the very least run the protocol for a varying number
of nodes. It is even better practice to test the different methods of the
protocol, as in Test Driven Development.
*/

import (
	"testing"
	"time"

	"github.com/dedis/student_17_bftcosi/protocol"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
)

func TestMain(m *testing.M) {
	log.MainTest(m)
}

// Tests a 2, 5 and 13-node system. It is good practice to test different
// sizes of trees to make sure your protocol is stable.
//TODO: delete this function once protocol working
func TestNode(t *testing.T) {
	local := onet.NewLocalTest()
	nodes := []int{2, 5, 13}
	for _, nbrNodes := range nodes {
		_, _, tree := protocol.GenTree(local, nbrNodes, 1, true)
		log.Lvl3(tree.Dump())

		pi, err := local.StartProtocol("Template", tree)
		if err != nil {
			t.Fatal("Couldn't start protocol:", err)
		}
		protocol := pi.(*protocol.Template)
		timeout := network.WaitRetry * time.Duration(network.MaxRetryConnect*nbrNodes*2) * time.Millisecond
		select {
		case children := <-protocol.ChildCount:
			log.Lvl2("Instance 1 is done")
			if children != nbrNodes {
				t.Fatal("Didn't get a child-cound of", nbrNodes, ", but got a child count of", children)
			}
		case <-time.After(timeout):
			t.Fatal("Didn't finish in time")
		}
		local.CloseAll()
	}
}

//tests the root of the tree
func TestGenTreeRoot(t *testing.T) {
	local := onet.NewLocalTest()

	nodes := []int{1, 2, 5, 13, 20}
	for _, nbrNodes := range nodes {
		_, _, tree := protocol.GenTree(local, nbrNodes, 12,  true)
		if tree.Root == nil {
			t.Fatal("Tree Root shouldn't be nil")
		}
		testNode(t, tree.Root, nil, tree)
		local.CloseAll()
	}
}

//tests the number of nodes of the tree
func TestGenTreeCount(t *testing.T) {
	local := onet.NewLocalTest()

	nodes := []int{1, 2, 5, 13, 20}
	for _, nbrNodes := range nodes {
		_, _, tree := protocol.GenTree(local, nbrNodes, 12,  true)
		if tree.Size() != nbrNodes {
			t.Fatal("The tree should contain", nbrNodes, "nodes, but contains", tree.Size(), "nodes")
		}
		testNode(t, tree.Root, nil, tree)
		local.CloseAll()
	}
}

//tests that the generated tree has the good number of shards
func TestGenTreeFirstLevel(t *testing.T) {
	local := onet.NewLocalTest()

	nodes := []int{1, 2, 5, 13, 20}
	nbrShards := 12
	for _, nbrNodes := range nodes {

		wantedShards := nbrShards
		if nbrNodes < nbrShards {
			wantedShards = nbrNodes-1
		}

		_, _, tree := protocol.GenTree(local, nbrNodes, nbrShards, true)
		actualShards := len(tree.Root.Children)
		if  actualShards != wantedShards {
			t.Fatal("There should be", wantedShards, "shards, but there is", actualShards, "shards")
		}
		local.CloseAll()
	}
}

//tests the complete tree
func TestGenTreeComplete(t *testing.T) {
	local := onet.NewLocalTest()

	nodes := []int{1, 2, 5, 13, 20}
	nbrShards := 12
	for _, nbrNodes := range nodes {

		_, _, tree := protocol.GenTree(local, nbrNodes, nbrShards, true)

		nodes_depth_2 := ((nbrNodes-1) / nbrShards) -1
		for _, n := range tree.Root.Children {
			if len(n.Children) < nodes_depth_2 || len(n.Children) > nodes_depth_2+1 {
				t.Fatal(nbrNodes, "node(s),", nbrShards,"shards: There should be",
					nodes_depth_2, "to", nodes_depth_2+1,"second level node(s)," +
					" but there is a shard with", len(n.Children), "second level node(s).")
			}
			testNode(t, n, tree.Root, tree)
			for _,m := range n.Children {
				if len(m.Children) > 0 {
					t.Fatal("the tree should be at most 2 level deep, but is not")
				}
				testNode(t, m, n, tree)
			}
		}
		local.CloseAll()
	}
}

//global tests to be performed on every node,
func testNode(t *testing.T, node, parent *onet.TreeNode, tree *onet.Tree) {
	if node.Parent != parent {
		t.Fatal("a node has not the right parent in the field \"parent\"")
	}
	addr, _ := tree.Roster.Search(node.ServerIdentity.ID)
	if addr == -1 {
		t.Fatal("a node in the tree is runing on a server that is not in the tree's roster")
	}
}