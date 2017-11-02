package protocol

import (
	"gopkg.in/dedis/onet.v1"
	"time"
	"fmt"
	"gopkg.in/dedis/crypto.v0/abstract"
	"errors"
	"gopkg.in/dedis/onet.v1/network"
)

func init() {
	network.RegisterMessage(Announcement{})
	network.RegisterMessage(Commitment{})
	network.RegisterMessage(Challenge{})
	network.RegisterMessage(Response{})

	onet.GlobalProtocolRegister(Name, NewProtocol)
}

type StartProtocolFunction func(name string, t *onet.Tree) (onet.ProtocolInstance, error)

func StartProtocol(startProtocol StartProtocolFunction, trees []*onet.Tree) ([][]byte, error){

	signatures := make([][]byte, len(trees))
	cosiProtocols := make([]*Cosi, len(trees))

	//start all protocols
	for i, tree := range trees {
		//start protocol
		pi, err := startProtocol(Name, tree)
		if err != nil {
			return nil, err
		}

		//get response
		cosiProtocols[i] = pi.(*Cosi)
	}

	//check all protocols
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

/*	The `NewProtocol` method is used to define the protocol and to register
	the channels where the messages will be received.
*/
func NewProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {

	nShards := len(n.Root().Children)
	if nShards < 1 { //to avoid divBy0 with one node tree
		nShards = 1
	}

	var list []abstract.Point
	for _, t := range n.Tree().List() {
		list = append(list, t.PublicAggregateSubTree)
	}

	c := &Cosi{
		TreeNodeInstance:    	n,
		List:               list,
		MinShardSize:        	n.Tree().Size()-1 / nShards,
		subleaderNotResponding: make(chan bool),
		FinalSignature:			make(chan []byte),
	}

	for _, channel := range []interface{}{&c.ChannelAnnouncement, &c.ChannelCommitment, &c.ChannelChallenge, &c.ChannelResponse} {
		err := c.RegisterChannel(channel)
		if err != nil {
			return nil, errors.New("couldn't register channel: " + err.Error())
		}
	}
	return c, nil
}