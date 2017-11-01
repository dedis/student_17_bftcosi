package protocol_tests

import (
	"testing"
	"time"

	"github.com/dedis/student_17_bftcosi/protocol"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
	"github.com/dedis/student_17_bftcosi/cosi"
	"gopkg.in/dedis/crypto.v0/abstract"
)

// Tests various trees configurations
func TestProtocol(t *testing.T) {
	//log.SetDebugVisible(3)

	local := onet.NewLocalTest()
	nodes := []int{2, 5, 13, 24}
	shards := []int{1, 2, 5}

	for _, nbrNodes := range nodes {
		for _, nbrShards := range shards {

			servers := local.GenServers(nbrNodes)
			roster := local.GenRosterFromHost(servers...)

			//generate tree
			err, tree := protocol.GenTree(roster, nbrNodes, nbrShards)
			if err != nil {
				t.Fatal("Error in tree generation:", err)
			}
			log.Lvl3(tree.Dump())

			//get public keys
			publics := make([]abstract.Point, 0)
			for _, n := range tree.List() {
				publics = append(publics, n.ServerIdentity.Public)
			}

			//start protocol
			pi, err := local.StartProtocol(protocol.Name, tree)
			if err != nil {
				t.Fatal("Couldn't start protocol:", err)
			}

			//get response
			protocol := pi.(*protocol.Cosi)
			timeout := network.WaitRetry * time.Duration(network.MaxRetryConnect*nbrNodes*2) * time.Millisecond
			select {
			case signature := <-protocol.FinalSignature:
				log.Lvl2("Instance is done")
				proposal := []byte{0xFF}
				err = cosi.Verify(protocol.Suite(), publics, proposal, signature, cosi.CompletePolicy{})
				if err != nil {
					t.Fatal("Didn't get a valid response aggregate:", err)
				} else {
					log.Lvl2("Signature correctly verified!")
				}
			case <-time.After(timeout):
				t.Fatal("Didn't finish in time")
			}
			local.CloseAll()
		}
	}
}