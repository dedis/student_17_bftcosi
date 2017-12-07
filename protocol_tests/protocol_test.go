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
)

// Tests various trees configurations
func TestProtocol(t *testing.T) {
	//log.SetDebugVisible(3)

	local := onet.NewLocalTest()
	nodes := []int{1, 2, 5, 13, 24}
	subtrees := []int{1, 2, 5}
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
				t.Fatal("Error in creation of protocol:", err)
			}
			cosiProtocol := pi.(*protocol.CosiRootNode)
			cosiProtocol.CreateProtocol = local.CreateProtocol
			cosiProtocol.Proposal = proposal
			cosiProtocol.NSubtrees = nSubtrees
			err = cosiProtocol.Start()
			if err != nil {
				t.Fatal("Error in starting of protocol:", err)
			}

			//get and verify signature
			err = getAndVerifySignature(cosiProtocol, publics, proposal)
			if err != nil {
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
	nodes := []int{5, 13, 24}
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

			//start protocol
			pi, err := local.CreateProtocol(protocol.ProtocolName, tree)
			if err != nil {
				t.Fatal("Error in creation of protocol:", err)
			}
			cosiProtocol := pi.(*protocol.CosiRootNode)
			cosiProtocol.CreateProtocol = local.CreateProtocol
			cosiProtocol.Proposal = proposal
			cosiProtocol.NSubtrees = nSubtrees
			cosiProtocol.ProtocolTimeout = protocol.DefaultProtocolTimeout / 10000

			//setup announcement interception
			subleaderServer := servers[1]
			subleaderServer.RegisterProcessorFunc(onet.ProtocolMsgID, func(e *network.Envelope) {
				if e.ServerIdentity.ID == tree.Root.ServerIdentity.ID {
					log.Lvl2("Dropped message from root")
				} else {
					local.Overlays[subleaderServer.ServerIdentity.ID].Process(e)
				}
			})

			err = cosiProtocol.Start()
			if err != nil {
				t.Fatal("Error in starting of protocol:", err)
			}

			//get and verify signature
			err = getAndVerifySignature(cosiProtocol, publics, proposal)
			if err != nil {
				t.Fatal(err)
			}

			local.CloseAll()
		}
	}
}

// Tests that the protocol throws errors with invalid configurations
func TestProtocolErrors(t *testing.T) { //TODO: implement protocol interruption
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
				t.Fatal("Error in creation of protocol:", err)
			}
			cosiProtocol := pi.(*protocol.CosiRootNode)
			//cosiProtocol.CreateProtocol = local.CreateProtocol
			cosiProtocol.Proposal = proposal
			cosiProtocol.NSubtrees = nSubtrees
			err = cosiProtocol.Start()
			if err == nil {
				t.Fatal("protocol should throw an error if called without create protocol function, but doesn't")
			}

			//missing proposal
			pi, err = local.CreateProtocol(protocol.ProtocolName, tree)
			if err != nil {
				t.Fatal("Error in creation of protocol:", err)
			}
			cosiProtocol = pi.(*protocol.CosiRootNode)
			cosiProtocol.CreateProtocol = local.CreateProtocol
			//cosiProtocol.Proposal = proposal
			cosiProtocol.NSubtrees = nSubtrees
			err = cosiProtocol.Start()
			if err == nil {
				t.Fatal("protocol should throw an error if called without a proposal, but doesn't")
			}

			local.CloseAll()
		}
	}
}

func getAndVerifySignature(cosiProtocol *protocol.CosiRootNode, publics []abstract.Point, proposal []byte) error {

	//get response
	var signature []byte
	select {
	case signature = <-cosiProtocol.FinalSignature:
		log.Lvl3("Instance is done")
	case <-time.After(protocol.DefaultProtocolTimeout):
		return fmt.Errorf("didn't get commitment in time")
	}

	//verify signature
	err := cosi.Verify(network.Suite, publics, proposal, signature, cosi.CompletePolicy{})
	if err != nil {
		return fmt.Errorf("didn't get a valid signature: %s", err)
	}
	log.Lvl2("Signature correctly verified!")
	return nil
}