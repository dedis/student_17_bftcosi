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

	nodes := []int{1, 2, 5, 20}
	subtrees := []int{1, 5, 12}
	for _, nbrNodes := range nodes {
		for _, nSubtrees := range subtrees {
			servers := local.GenServers(nbrNodes)

			trees, err := protocol.GenTrees(servers, local.GenRosterFromHost, nbrNodes, nSubtrees)
			if err != nil {
				t.Fatal("Error in tree generation:", err)
			}
			root := trees[0].Root.ServerIdentity
			for _, tree := range trees {
				if tree.Root == nil { //TODO: do this test in testNode?
					t.Fatal("Tree Root shouldn't be nil")
				}
				if tree.Root.ServerIdentity != root {
					t.Fatal("Tree Root should be the same for all trees, but isn't")
				}
				testNode(t, tree.Root, nil, tree)
			}
			local.CloseAll()
		}
	}
}

//tests the number of nodes of the tree
func TestGenTreeCount(t *testing.T) {
	local := onet.NewLocalTest()

	nodes := []int{1, 2, 5, 20}
	subtrees := []int{1, 5, 12}
	for _, nNodes := range nodes {
		for _, nSubtrees := range subtrees {
			servers := local.GenServers(nNodes)

			trees, err := protocol.GenTrees(servers, local.GenRosterFromHost, nNodes, nSubtrees)
			if err != nil {
				t.Fatal("Error in tree generation:", err)
			}
			totalNodes := 1
			expectedNodesPerTree := (nNodes-1)/len(trees) + 1
			for i, tree := range trees {
				if tree.Size() != expectedNodesPerTree && tree.Size() != expectedNodesPerTree+1 {
					t.Fatal("The subtree", i, "should contain", expectedNodesPerTree, "nodes, but contains", tree.Size(), "nodes")
				}
				totalNodes += tree.Size() - 1 //to account for shared leader
			}
			if totalNodes != nNodes {
				t.Fatal("Trees should in total contain", nNodes, "nodes, but they contain", totalNodes, "nodes")
			}
			local.CloseAll()
		}
	}
}

//tests that the generated tree has the good number of subtrees
func TestGenTreeSubtrees(t *testing.T) {
	local := onet.NewLocalTest()

	nodes := []int{1, 2, 5, 20}
	subtrees := []int{1, 5, 12}
	for _, nNodes := range nodes {
		for _, nSubtrees := range subtrees {

			wantedSubtrees := nSubtrees
			if nNodes <= nSubtrees {
				wantedSubtrees = nNodes - 1
				if wantedSubtrees < 1 {
					wantedSubtrees = 1
				}
			}

			servers := local.GenServers(nNodes)

			trees, err := protocol.GenTrees(servers, local.GenRosterFromHost, nNodes, nSubtrees)
			if err != nil {
				t.Fatal("Error in tree generation:", err)
			}
			actualSubtrees := len(trees)
			if actualSubtrees != wantedSubtrees {
				t.Fatal("There should be", wantedSubtrees, "subtrees, but there is", actualSubtrees, "subtrees")
			}
			local.CloseAll()
		}
	}
}

//tests the second and third level of all trees
func TestGenTreeComplete(t *testing.T) {
	local := onet.NewLocalTest()

	nodes := []int{1, 2, 5, 20}
	subtrees := []int{1, 5, 12}
	for _, nNodes := range nodes {
		for _, nSubtrees := range subtrees {

			servers := local.GenServers(nNodes)

			trees, err := protocol.GenTrees(servers, local.GenRosterFromHost, nNodes, nSubtrees)
			if err != nil {
				t.Fatal("Error in tree generation:", err)
			}

			nodesDepth2 := ((nNodes - 1) / nSubtrees) - 1
			for _, tree := range trees {
				if tree.Size() < 2 {
					continue
				}
				subleader := tree.Root.Children[0]
				if len(subleader.Children) < nodesDepth2 || len(subleader.Children) > nodesDepth2+1 {
					t.Fatal(nNodes, "node(s),", nSubtrees, "subtrees: There should be",
						nodesDepth2, "to", nodesDepth2+1, "second level node(s),"+
							" but there is a subtree with", len(subleader.Children), "second level node(s).")
				}
				testNode(t, subleader, tree.Root, tree)
				for _, m := range subleader.Children {
					if len(m.Children) > 0 {
						t.Fatal("the tree should be at most 2 level deep, but is not")
					}
					testNode(t, m, subleader, tree)
				}
			}
			local.CloseAll()
		}
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

		trees, err := protocol.GenTrees(servers, local.GenRosterFromHost, negativeNumber, positiveNumber)
		if err == nil {
			t.Fatal("the GenTree function should throw an error" +
				" with negative number of nodes, but doesn't")
		}
		if trees != nil {
			t.Fatal("the GenTree function should return a nil tree" +
			" with errors, but doesn't")
		}

		trees, err = protocol.GenTrees(servers, local.GenRosterFromHost, positiveNumber, negativeNumber)
		if err == nil {
			t.Fatal("the GenTree function should throw an error" +
				" with negative number of subtrees, but doesn't")
		}
		if trees != nil {
			t.Fatal("the GenTree function should return a nil tree" +
				" with errors, but doesn't")
		}

		local.CloseAll()
	}
}

//tests that the GenTree function returns roster errors correctly
func TestGenTreeRosterErrors(t *testing.T) {
	local := onet.NewLocalTest()

	trees, err := protocol.GenTrees(nil, local.GenRosterFromHost, 12, 3)
	if err == nil {
		t.Fatal("the GenTree function should throw an error" +
			" with an nil list of servers, but doesn't")
	}
	if trees != nil {
		t.Fatal("the GenTree function should return a nil tree" +
			" with errors, but doesn't")
	}

	servers := local.GenServers(2)

	trees, err = protocol.GenTrees(servers, local.GenRosterFromHost, 12, 3)
	if err == nil {
		t.Fatal("the GenTree function should throw an error" +
			" with a list of servers smaller than the number of nodes, but doesn't")
	}
	if trees != nil {
		t.Fatal("the GenTree function should return a nil tree" +
			" with errors, but doesn't")
	}

	local.CloseAll()
}

//tests that the GenTree function uses as many different servers from the roster as possible
func TestGenTreeUsesWholeRoster(t *testing.T) {
	local := onet.NewLocalTest()

	servers := []int{5, 13, 20}
	nNodes := 5
	for _, nServers := range servers {

		servers := local.GenServers(nServers)

		trees, err := protocol.GenTrees(servers, local.GenRosterFromHost, nNodes, 4)
		if err != nil {
			t.Fatal("Error in tree generation:", err)
		}

		serverSet := make(map[*network.ServerIdentity]bool)
		expectedUsedServers := nNodes
		if nServers < nNodes {
			expectedUsedServers = nServers
		}

		//get all the used serverIdentities
		for _, tree := range trees {
			serverSet[tree.Root.ServerIdentity] = true
			if tree.Size() > 1 {
				subleader := tree.Root.Children[0]
				serverSet[subleader.ServerIdentity] = true
				for _, m := range subleader.Children {
					serverSet[m.ServerIdentity] = true
				}
			}
		}

		if len(serverSet) != expectedUsedServers {
			t.Fatal("the generated tree should use", expectedUsedServers,
				"different servers but uses", len(serverSet))
		}


		local.CloseAll()
	}
}
