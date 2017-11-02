package protocol

import (
	"errors"
	"fmt"
	"time"

	"gopkg.in/dedis/crypto.v0/abstract"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/network"
)

func init() {
	network.RegisterMessage(Announcement{})
	network.RegisterMessage(Commitment{})
	network.RegisterMessage(Challenge{})
	network.RegisterMessage(Response{})

	onet.GlobalProtocolRegister(Name, NewProtocol)
}

//TODO: see if necessary
type StartProtocolFunction func(name string, t *onet.Tree) (onet.ProtocolInstance, error)
type RosterGenerator func(...*onet.Server) *onet.Roster

func StartProtocol(servers []*onet.Server, nNodes, nSubtrees int, rosterGenerator RosterGenerator, startProtocol StartProtocolFunction) ([][]byte, error){

	//generate trees
	trees, err := GenTrees(servers, rosterGenerator, nNodes, nSubtrees)
	if err != nil {
		return nil, fmt.Errorf("Error in tree generation:", err)
	}


	//start all protocols
	cosiProtocols := make([]*Cosi, len(trees))
	for i, tree := range trees {
		//start protocol
		pi, err := startProtocol(Name, tree)
		if err != nil {
			return nil, err
		}

		//get response
		cosiProtocols[i] = pi.(*Cosi)
	}

	//get all signatures
	signatures := make([][]byte, len(trees))
	timeout := 4*Timeout
	for i, cosiProtocol := range cosiProtocols {
		protocol := cosiProtocol
		for signatures[i] == nil {
			select {
			case _ = <-protocol.subleaderNotResponding:
				//restart protocol
				pi, err := startProtocol(Name, trees[i])
				if err != nil {
					return nil, err
				}
				protocol = pi.(*Cosi)
			case signature := <-protocol.FinalSignature:
				signatures[i] = signature
			case <-time.After(timeout):
				return nil, fmt.Errorf("Didn't finish in time")
			}
		}
	}

	return signatures, nil
}

// The `NewProtocol` method is used to define the protocol and to register
// the channels where the messages will be received.
func NewProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {

	nSubtrees := len(n.Root().Children)
	if nSubtrees < 1 { //to avoid divBy0 with one node tree
		nSubtrees = 1
	}

	var list []abstract.Point
	for _, t := range n.Tree().List() {
		list = append(list, t.PublicAggregateSubTree)
	}

	c := &Cosi{
		TreeNodeInstance:       n,
		List:                   list,
		MinSubtreeSize:         n.Tree().Size()-1 /nSubtrees +1,
		subleaderNotResponding: make(chan bool),
		FinalSignature:         make(chan []byte),
	}

	for _, channel := range []interface{}{&c.ChannelAnnouncement, &c.ChannelCommitment, &c.ChannelChallenge, &c.ChannelResponse} {
		err := c.RegisterChannel(channel)
		if err != nil {
			return nil, errors.New("couldn't register channel: " + err.Error())
		}
	}
	return c, nil
}