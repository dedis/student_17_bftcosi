package protocol_tests

import (
	"testing"
	"time"

	"github.com/dedis/student_17_bftcosi/protocol"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
)

// Tests a 2, 5 and 13-node system. It is good practice to test different
// sizes of trees to make sure your protocol is stable.
func TestProtocol(t *testing.T) {
	local := onet.NewLocalTest()
	nodes := []int{2, 5, 13}

	for _, nbrNodes := range nodes {

		servers := local.GenServers(nbrNodes)
		roster := local.GenRosterFromHost(servers...)

		err, tree := protocol.GenTree(roster, nbrNodes, 1)
		if err != nil {
			t.Fatal("Error in tree generation:", err)
		}
		log.Lvl3(tree.Dump())

		pi, err := local.StartProtocol(protocol.Name, tree)
		if err != nil {
			t.Fatal("Couldn't start protocol:", err)
		}
		protocol := pi.(*protocol.Cosi)
		timeout := network.WaitRetry * time.Duration(network.MaxRetryConnect*nbrNodes*2) * time.Millisecond
		select {
		case aggResponse := <-protocol.AggregateResponse:
			log.Lvl2("Instance 1 is done")
			if aggResponse.CosiReponse == nil {
				t.Fatal("Didn't get a valid response aggregate")
			}
		case <-time.After(timeout):
			t.Fatal("Didn't finish in time")
		}
		local.CloseAll()
	}
}