package protocol_tests

import (
	"testing"

	"github.com/dedis/student_17_bftcosi/protocol"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
	"github.com/dedis/student_17_bftcosi/cosi"
	"gopkg.in/dedis/crypto.v0/abstract"
	"time"
	"fmt"
	"reflect"
)

// Tests various trees configurations
func TestProtocol(t *testing.T) {
	//log.SetDebugVisible(3)

	local := onet.NewLocalTest()
	nodes := []int{1, 2, 5, 13, 24, 100}
	subtrees := []int{1, 2, 5, 9}
	proposal := []byte{0xFF}

	for _, nNodes := range nodes {
		for _, nSubtrees := range subtrees {
			log.Lvl2("test asking for",nNodes, "nodes and", nSubtrees, "subtrees")

			_, _, tree := local.GenTree(nNodes, false)

			//get public keys
			publics := make([]abstract.Point, tree.Size())
			for i, node := range tree.List() {
				publics[i] = node.ServerIdentity.Public
			}

			//start protocol
			pi, err := local.CreateProtocol(protocol.ProtocolName, tree)
			if err != nil {
				local.CloseAll()
				t.Fatal("Error in creation of protocol:", err)
			}
			cosiProtocol := pi.(*protocol.CoSiRootNode)
			cosiProtocol.CreateProtocol = local.CreateProtocol
			cosiProtocol.Proposal = proposal
			cosiProtocol.NSubtrees = nSubtrees
			err = cosiProtocol.Start()
			if err != nil {
				local.CloseAll()
				t.Fatal("Error in starting of protocol:", err)
			}

			//get and verify signature
			err = getAndVerifySignature(cosiProtocol, publics, proposal, cosi.CompletePolicy{})
			if err != nil {
				local.CloseAll()
				t.Fatal(err)
			}

			local.CloseAll()
		}
	}
}

// Tests unresponsive leaves in various tree configurations
func TestUnresponsiveLeafs(t *testing.T) {
	//log.SetDebugVisible(3)

	local := onet.NewLocalTest()
	nodes := []int{3, 13, 24}
	subtrees := []int{1, 2}
	proposal := []byte{0xFF}

	for _, nNodes := range nodes {
		for _, nSubtrees := range subtrees {
			log.Lvl2("test asking for",nNodes, "nodes and", nSubtrees, "subtrees")

			servers, _, tree := local.GenTree(nNodes, false)

			//get public keys
			publics := make([]abstract.Point, tree.Size())
			for i, node := range tree.List() {
				publics[i] = node.ServerIdentity.Public
			}

			//create protocol
			pi, err := local.CreateProtocol(protocol.ProtocolName, tree)
			if err != nil {
				local.CloseAll()
				t.Fatal("Error in creation of protocol:", err)
			}
			cosiProtocol := pi.(*protocol.CoSiRootNode)
			cosiProtocol.CreateProtocol = local.CreateProtocol
			cosiProtocol.Proposal = proposal
			cosiProtocol.NSubtrees = nSubtrees
			cosiProtocol.LeavesTimeout = protocol.DefaultLeavesTimeout / 5000

			//find first subtree leaves servers based on GenTree function
			leafsServerIdentities, err := protocol.GetLeafsIDs(tree, nNodes, nSubtrees)
			if err != nil {
				t.Fatal(err)
			}
			failing := len(leafsServerIdentities) / 3 //we render unresponsive one third of leafs
			failingLeafsServerIdentities := leafsServerIdentities[:failing]
			firstLeavesServers := make([]*onet.Server, 0)
			for _, s := range servers {
				for _, l := range failingLeafsServerIdentities {
					if s.ServerIdentity.ID == l {
						firstLeavesServers = append(firstLeavesServers, s)
						break
					}
				}
			}

			//setup message interception on all first subtree leaves
			for _, l := range firstLeavesServers {
				l.RegisterProcessorFunc(onet.ProtocolMsgID, func(e *network.Envelope) {
					log.Lvl3("Dropped message")
				})
			}

			//start protocol
			err = cosiProtocol.Start()
			if err != nil {
				local.CloseAll()
				t.Fatal("error in starting of protocol:", err)
			}

			//get and verify signature
			threshold := nNodes - failing
			err = getAndVerifySignature(cosiProtocol, publics, proposal, cosi.ThresholdPolicy{T:threshold})
			if err != nil {
				local.CloseAll()
				t.Fatal(err)
			}

			local.CloseAll()
		}
	}
}

// Tests unresponsive subleaders in various tree configurations
func TestUnresponsiveSubleader(t *testing.T) {
	//log.SetDebugVisible(3)

	local := onet.NewLocalTest()
	nodes := []int{6, 13, 24}
	subtrees := []int{1, 2}
	proposal := []byte{0xFF}

	for _, nNodes := range nodes {
		for _, nSubtrees := range subtrees {
			log.Lvl2("test asking for",nNodes, "nodes and", nSubtrees, "subtrees")

			servers, _, tree := local.GenTree(nNodes, false)

			//get public keys
			publics := make([]abstract.Point, tree.Size())
			for i, node := range tree.List() {
				publics[i] = node.ServerIdentity.Public
			}

			//create protocol
			pi, err := local.CreateProtocol(protocol.ProtocolName, tree)
			if err != nil {
				local.CloseAll()
				t.Fatal("Error in creation of protocol:", err)
			}
			cosiProtocol := pi.(*protocol.CoSiRootNode)
			cosiProtocol.CreateProtocol = local.CreateProtocol
			cosiProtocol.Proposal = proposal
			cosiProtocol.NSubtrees = nSubtrees
			cosiProtocol.SubleaderTimeout = protocol.DefaultSubleaderTimeout / 7000

			//find first subleader server based on genTree function
			subleaderIds, err := protocol.GetSubleaderIDs(tree, nNodes, nSubtrees)
			if err != nil {
				local.CloseAll()
				t.Fatal(err)
			} else if len(subleaderIds) < 1 {
				local.CloseAll()
				t.Fatal("found no subleader in generated tree with ", nNodes, "nodes and", nSubtrees, "subtrees")
			}
			var firstSubleaderServer *onet.Server
			for _, s := range servers {
				if s.ServerIdentity.ID == subleaderIds[0] {
					firstSubleaderServer = s
					break
				}
			}

			//setup message interception on first subleader //TODO: intercept only root's announcements
			firstSubleaderServer.RegisterProcessorFunc(onet.ProtocolMsgID, func(e *network.Envelope) {
				if e.ServerIdentity.ID == tree.Root.ServerIdentity.ID {
					_, msg, err := network.Unmarshal(e.Msg.(*onet.ProtocolMsg).MsgSlice)
					if err != nil {
						local.CloseAll()
						t.Fatal("error while unmarshelling message", err)
					}
					log.Lvl2(firstSubleaderServer.Address(), "Dropped message from root of type:", reflect.TypeOf(msg))
				} else {
					local.Overlays[firstSubleaderServer.ServerIdentity.ID].Process(e)
				}
			})

			//start protocol
			err = cosiProtocol.Start()
			if err != nil {
				local.CloseAll()
				t.Fatal("Error in starting of protocol:", err)
			}

			//get and verify signature
			err = getAndVerifySignature(cosiProtocol, publics, proposal, cosi.CompletePolicy{})
			if err != nil {
				local.CloseAll()
				t.Fatal(err)
			}

			local.CloseAll()
		}
	}
}

// Tests that the protocol throws errors with invalid configurations
func TestProtocolErrors(t *testing.T) {
	//log.SetDebugVisible(3)

	local := onet.NewLocalTest()
	nodes := []int{1, 2, 5, 13, 24}
	subtrees := []int{1, 2, 5}
	proposal := []byte{0xFF}

	for _, nNodes := range nodes {
		for _, nSubtrees := range subtrees {
			log.Lvl2("test asking for",nNodes, "nodes and", nSubtrees, "subtrees")

			_, _, tree := local.GenTree(nNodes, false)

			//missing create protocol function
			pi, err := local.CreateProtocol(protocol.ProtocolName, tree)
			if err != nil {
				local.CloseAll()
				t.Fatal("Error in creation of protocol:", err)
			}
			cosiProtocol := pi.(*protocol.CoSiRootNode)
			//cosiProtocol.CreateProtocol = local.CreateProtocol
			cosiProtocol.Proposal = proposal
			cosiProtocol.NSubtrees = nSubtrees
			err = cosiProtocol.Start()
			if err == nil {
				local.CloseAll()
				t.Fatal("protocol should throw an error if called without create protocol function, but doesn't")
			}

			//missing proposal
			pi, err = local.CreateProtocol(protocol.ProtocolName, tree)
			if err != nil {
				local.CloseAll()
				t.Fatal("Error in creation of protocol:", err)
			}
			cosiProtocol = pi.(*protocol.CoSiRootNode)
			cosiProtocol.CreateProtocol = local.CreateProtocol
			//cosiProtocol.Proposal = proposal
			cosiProtocol.NSubtrees = nSubtrees
			err = cosiProtocol.Start()
			if err == nil {
				local.CloseAll()
				t.Fatal("protocol should throw an error if called without a proposal, but doesn't")
			}

			local.CloseAll()
		}
	}
}

func getAndVerifySignature(cosiProtocol *protocol.CoSiRootNode, publics []abstract.Point,
	proposal []byte, policy cosi.Policy) error {

	//get response
	var signature []byte
	select {
	case signature = <-cosiProtocol.FinalSignature:
		log.Lvl3("Instance is done")
	case <-time.After(protocol.DefaultProtocolTimeout):
		return fmt.Errorf("didn't get commitment in time")
	}

	//verify signature
	err := cosi.Verify(network.Suite, publics, proposal, signature, policy)
	if err != nil {
		return fmt.Errorf("didn't get a valid signature: %s", err)
	}
	log.Lvl2("Signature correctly verified!")
	return nil
}