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

			//generate trees
			trees, err := protocol.GenTrees(servers, local.GenRosterFromHost, nbrNodes, nbrShards)
			if err != nil {
				t.Fatal("Error in tree generation:", err)
			}

			//get public keys
			publics := make([][]abstract.Point, len(trees))
			for i, tree := range trees {
				publics[i] = make([]abstract.Point, len(tree.List()))
				for j, n := range tree.List() {
					publics[i][j] = n.ServerIdentity.Public
				}
			}

			//start protocol
			channelsSignature, err := protocol.SuperProtocol(local.StartProtocol, trees)
			if err != nil {
				t.Fatal("Error in protocol:", err)
			}

			//get responses
			for i, channelSignature := range channelsSignature {
				timeout := network.WaitRetry * time.Duration(network.MaxRetryConnect*nbrNodes*2) * time.Millisecond
				select {
				case signature := <-channelSignature:
					log.Lvl2("Instance is done")
					proposal := []byte{0xFF}
					err = cosi.Verify(network.Suite, publics[i], proposal, signature, cosi.CompletePolicy{})
					if err != nil {
						t.Fatal("Didn't get a valid response aggregate:", err)
					} else {
						log.Lvl2("Signature correctly verified!")
					}
				case <-time.After(timeout):
					t.Fatal("Didn't finish in time")
				}
			}
			local.CloseAll()
		}
	}
}
