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

			//get response
			var signature []byte
			select {
				case signature = <-cosiProtocol.FinalSignature:
					log.Lvl3("Instance is done")
				case <-time.After(protocol.DefaultProtocolTimeout):
					t.Fatal("Didn't get commitment in time")
			}

			//verify signature
			err = cosi.Verify(network.Suite, publics, proposal, signature, cosi.CompletePolicy{})
			if err != nil {
				t.Fatal("Error while verifying signature:", err)
			}
			log.Lvl2("Signature correctly verified!")
			local.CloseAll()
		}
	}
}

// Tests unresponsive subleaders in various tree configurations
func TestUnresponsiveSubleader(t *testing.T) {
	//log.SetDebugVisible(3)

	local := onet.NewLocalTest()
	nodes := []int{3, 5, 13, 24}
	subtrees := []int{1, 2, 5}
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
			AnnouncementDropped := false
			subleaderServer := servers[1]
			subleaderServer.RegisterProcessorFunc(onet.ProtocolMsgID, func(e *network.Envelope) {

				//get message
				protoMsg := e.Msg.(*onet.ProtocolMsg)
				_, msg, err := network.Unmarshal(protoMsg.MsgSlice)
				if err != nil {
					t.Fatal("error while unmarshaling a message:", err)
				}

				//ignore the first announcement
				switch msg.(type) { //TODO: ask for whether or not we can compare protoMsg.From with server[0] id
				case *protocol.Announcement:
					if AnnouncementDropped {
						local.Overlays[subleaderServer.ServerIdentity.ID].Process(e)
					} else {
						log.Lvl2("Dropped first announcement")
						AnnouncementDropped = true
					}
				default:
					local.Overlays[subleaderServer.ServerIdentity.ID].Process(e)
				}
			})


			err = cosiProtocol.Start()
			if err != nil {
				t.Fatal("Error in starting of protocol:", err)
			}

			//get response
			var signature []byte
			select {
			case signature = <-cosiProtocol.FinalSignature:
				log.Lvl3("Instance is done")
			case <-time.After(protocol.DefaultProtocolTimeout):
				t.Fatal("Didn't get commitment in time")
			}

			//verify signature
			err = cosi.Verify(network.Suite, publics, proposal, signature, cosi.CompletePolicy{})
			if err != nil {
				t.Fatal("Didn't get a valid response aggregate:", err)
			}
			log.Lvl2("Signature correctly verified!")
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