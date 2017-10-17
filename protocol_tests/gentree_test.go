package protocol_tests

/*
The test-file should at the very least run the protocol for a varying number
of nodes. It is even better practice to test the different methods of the
protocol, as in Test Driven Development.
*/

import (
	"testing"

	"github.com/dedis/student_17_bftcosi/protocol"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
)

func TestMain(m *testing.M) {
	log.MainTest(m)
}

//tests the root of the tree
func TestGenTreeRoot(t *testing.T) {
	local := onet.NewLocalTest()

	nodes := []int{1, 2, 5, 13, 20}
	for _, nbrNodes := range nodes {
		servers := local.GenServers(nbrNodes)
		roster := local.GenRosterFromHost(servers...)

		err, tree := protocol.GenTree(roster, nbrNodes, 12)
		if err != nil {
			t.Fatal("Error in tree generation:", err)
		}
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
		servers := local.GenServers(nbrNodes)
		roster := local.GenRosterFromHost(servers...)

		err, tree := protocol.GenTree(roster, nbrNodes, 12)
		if err != nil {
			t.Fatal("Error in tree generation:", err)
		}
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

		servers := local.GenServers(nbrNodes)
		roster := local.GenRosterFromHost(servers...)

		err, tree := protocol.GenTree(roster, nbrNodes, nbrShards)
		if err != nil {
			t.Fatal("Error in tree generation:", err)
		}
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

		servers := local.GenServers(nbrNodes)
		roster := local.GenRosterFromHost(servers...)

		err, tree := protocol.GenTree(roster, nbrNodes, nbrShards)
		if err != nil {
			t.Fatal("Error in tree generation:", err)
		}

		nodesDepth2 := ((nbrNodes-1) / nbrShards) -1
		for _, n := range tree.Root.Children {
			if len(n.Children) < nodesDepth2 || len(n.Children) > nodesDepth2+1 {
				t.Fatal(nbrNodes, "node(s),", nbrShards,"shards: There should be",
					nodesDepth2, "to", nodesDepth2+1,"second level node(s)," +
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


//tests that the GenTree function returns errors correctly
func TestGenTreeErrors(t *testing.T) {
	local := onet.NewLocalTest()

	negativeNumbers := []int{0, -1, -2, -12, -34}
	positiveNumber := 12
	for _, negativeNumber := range negativeNumbers {

		servers := local.GenServers(positiveNumber)
		roster := local.GenRosterFromHost(servers...)

		err, tree := protocol.GenTree(roster, negativeNumber, positiveNumber)
		if err == nil {
			t.Fatal("the GenTree function should throw an error" +
				" with negative number of nodes, but doesn't")
		}
		if tree != nil {
			t.Fatal("the GenTree function should return a nil tree" +
			" with errors, but doesn't")
		}

		err, tree = protocol.GenTree(roster, positiveNumber, negativeNumber)
		if err == nil {
			t.Fatal("the GenTree function should throw an error" +
				" with negative number of shards, but doesn't")
		}
		if tree != nil {
			t.Fatal("the GenTree function should return a nil tree" +
				" with errors, but doesn't")
		}

		local.CloseAll()
	}
}

//tests that the GenTree function returns roster errors correctly
func TestGenTreeRosterErrors(t *testing.T) {
	local := onet.NewLocalTest()

	err, tree := protocol.GenTree(nil, 12, 3)
	if err == nil {
		t.Fatal("the GenTree function should throw an error" +
			" with an nil roster, but doesn't")
	}
	if tree != nil {
		t.Fatal("the GenTree function should return a nil tree" +
			" with errors, but doesn't")
	}

	servers := local.GenServers(0)
	roster := local.GenRosterFromHost(servers...)

	err, tree = protocol.GenTree(roster, 12, 3)
	if err == nil {
		t.Fatal("the GenTree function should throw an error" +
			" with an empty roster, but doesn't")
	}
	if tree != nil {
		t.Fatal("the GenTree function should return a nil tree" +
			" with errors, but doesn't")
	}

	local.CloseAll()
}

//tests that the GenTree function uses as many different servers from the roster as possible
func TestGenTreeUsesWholeRoster(t *testing.T) {
	local := onet.NewLocalTest()

	servers := []int{1, 2, 5, 13, 20}
	nbrNodes := 5
	for _, nbrServers := range servers {

		servers := local.GenServers(nbrServers)
		roster := local.GenRosterFromHost(servers...)

		err, tree := protocol.GenTree(roster, nbrNodes, 4)
		if err != nil {
			t.Fatal("Error in tree generation:", err)
		}

		serverSet := make(map[*network.ServerIdentity]bool)
		expectedUsedServers := nbrNodes
		if nbrServers < nbrNodes {
			expectedUsedServers = nbrServers
		}

		//get all the used serverIdentities
		serverSet[tree.Root.ServerIdentity] = true
		for _, n := range tree.Root.Children {
			serverSet[n.ServerIdentity] = true
			for _, m := range n.Children {
				serverSet[m.ServerIdentity] = true
			}
		}

		if len(serverSet) != expectedUsedServers {
			t.Fatal("the generated tree should use", expectedUsedServers,
				"different servers but uses", len(serverSet))
		}


		local.CloseAll()
	}
}