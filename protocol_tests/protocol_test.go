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
				t.Fatal("Didn't get a valid response aggregate:", err)
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
			log.Lvl2(nNodes, nSubtrees)

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

			//define intercept message
			AnnouncementDropped := false
			servers[1].RegisterProcessorFunc(onet.ProtocolMsgID, func(e *network.Envelope) {
				// protoMsg holds also To and From fields that can help decide
				// whether a message should be sent or not.
				protoMsg := e.Msg.(*onet.ProtocolMsg)
				_, msg, err := network.Unmarshal(protoMsg.MsgSlice)
				if err != nil {
					t.Fatal("error while unmarshaling a message:", err)
				}

				// Finally give the message back to onet. If this last call is not
				// made, the message is dropped.

				switch msg.(type) { //TODO: ask for wether or not we can get server[0] id
				case *protocol.Announcement:
					if AnnouncementDropped {
						local.Overlays[servers[1].ServerIdentity.ID].Process(e)
					} else {
						log.Lvlf2("Dropped first announcement")
					}
					AnnouncementDropped = true
				default:
					local.Overlays[servers[1].ServerIdentity.ID].Process(e)
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