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
			select {
				case signature := <-cosiProtocol.FinalSignature:
					log.Lvl2("Instance is done")
					err = cosi.Verify(network.Suite, publics, proposal, signature, cosi.CompletePolicy{})
					if err != nil {
						t.Fatal("Didn't get a valid response aggregate:", err)
					}
				case <-time.After(protocol.Timeout):
					t.Fatal("Didn't get commitment in time")
			}

			log.Lvl2("Signature correctly verified!")
			local.CloseAll()
		}
	}
}
