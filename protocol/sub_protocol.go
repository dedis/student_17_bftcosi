package protocol

import (
	"time"

	"github.com/dedis/student_17_bftcosi/cosi"
	"gopkg.in/dedis/crypto.v0/abstract"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
	"fmt"
	"errors"
)

// CosiSubProtocolNode holds the different channels used to receive the different protocol messages.
type CosiSubProtocolNode struct {
	*onet.TreeNodeInstance
	Publics					[]abstract.Point
	Proposal				[]byte
	SubleaderTimeout		time.Duration //only defined for the root
	hasStopped				bool //used since Shutdown can be called multiple time

	//protocol/subprotocol channels
	subleaderNotResponding chan bool
	subCommitment		   chan StructCommitment
	subResponse            chan StructResponse

	//internodes channels
	ChannelAnnouncement    chan StructAnnouncement
	ChannelCommitment      chan []StructCommitment
	ChannelChallenge       chan StructChallenge
	ChannelResponse        chan []StructResponse
}

// The `NewSubProtocol` method is used to define the subprotocol and to register
// the channels where the messages will be received.
func NewSubProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {

	c := &CosiSubProtocolNode{
		TreeNodeInstance:       n,
		hasStopped:				false,
	}

	if n.IsRoot() {
		c.subleaderNotResponding = make(chan bool)
		c.subCommitment	= make(chan StructCommitment)
		c.subResponse =	make(chan StructResponse)
	}

	for _, channel := range []interface{}{&c.ChannelAnnouncement, &c.ChannelCommitment, &c.ChannelChallenge, &c.ChannelResponse} {
		err := c.RegisterChannel(channel)
		if err != nil {
			return nil, errors.New("couldn't register channel: " + err.Error())
		}
	}
	err := c.RegisterHandler(c.HandleStop)
	if err != nil {
		return nil, errors.New("couldn't register stop handler: " + err.Error())
	}
	return c, nil
}

func (p *CosiSubProtocolNode) Shutdown() error {
	if !p.hasStopped {
		close(p.ChannelAnnouncement)
		close(p.ChannelCommitment)
		close(p.ChannelChallenge)
		close(p.ChannelResponse)
		p.hasStopped = true
	}
	return nil
}

//Dispatch() is the main method of the subprotocol, running on each node and handling the messages in order
func (p *CosiSubProtocolNode) Dispatch() error {

	// ----- Announcement -----
	announcement, channelOpen := <-p.ChannelAnnouncement
	if !channelOpen {
		return nil
	}
	log.Lvl3(p.ServerIdentity().Address, "received announcement")
	p.Publics = announcement.Publics
	err := p.SendToChildren(&announcement.Announcement)
	if err != nil {
		return err
	}

	// ----- Commitment -----
	if p.IsLeaf() {
		p.ChannelCommitment <- make([]StructCommitment, 0)
	}
	commitments := make([]StructCommitment, 0)
	if p.IsRoot() {
		select {
		case commitments = <-p.ChannelCommitment:
		case <-time.After(p.SubleaderTimeout):
			p.subleaderNotResponding <- true
			return nil
		}
	} else {
		commitments = <-p.ChannelCommitment
	}
	log.Lvl3(p.ServerIdentity().Address, "received commitment")

 	var secret abstract.Scalar

 	// if root, send commitment to super-protocol
	if p.IsRoot() {
		if len(commitments) != 1 {
			return fmt.Errorf("root node in subprotocol should have received 1 commitment," +
				"but received %d", len(commitments))
		}
		p.subCommitment <- commitments[0]

	// if not root, compute personal commitment and send to parent
	} else {
		var commitment abstract.Point
		var mask *cosi.Mask
		secret, commitment, mask, err = generatePersonnalCommitment(p.TreeNodeInstance, p.Publics, commitments)
		if err != nil {
			return err
		}
		err = p.SendToParent(&Commitment{commitment, mask.Mask()})
		if err != nil {
			return err
		}
	}

	// ----- Challenge -----
	challenge := <-p.ChannelChallenge
	log.Lvl3(p.ServerIdentity().Address, "received challenge")
	err = p.SendToChildren(&challenge.Challenge)
	if err != nil {
		return err
	}

	// ----- Response -----

	//get response
	if p.IsLeaf() {
		p.ChannelResponse <- make([]StructResponse, 0)
	}
	responses := <-p.ChannelResponse
	log.Lvl3(p.ServerIdentity().Address, "received response")

	//if root, send response to super-protocol
	if p.IsRoot() {
		if len(responses) != 1 {
			return fmt.Errorf("root node in subprotocol should have received 1 response," +
				"but received %d", len(commitments))
		}
		p.subResponse <- responses[0]

	// if not root, generate own response and send to parent
	} else {
		response, err := generateResponse(p.TreeNodeInstance, responses, secret, challenge.Challenge.CosiChallenge)
		if err != nil {
			return err
		}
		err = p.SendToParent(&Response{response})
		if err != nil {
			return err
		}
	}

	//TODO: see if should stop node or be ready for another proposal
	return nil
}

//HandleStop is called when a Stop message is send to this node.
// It broadcasts the message and stops the node
func (p *CosiSubProtocolNode) HandleStop(stop StructStop) error {
	defer p.Done()
	if p.IsRoot() {
		p.Broadcast(&stop.Stop)
	}
	return nil
}

// Start is done only by root and starts the subprotocol
func (p *CosiSubProtocolNode) Start() error {
	log.Lvl3("Starting subCoSi")
	if p.Proposal == nil {
		return fmt.Errorf("subprotocol started without any proposal set")
	} else if p.Publics == nil || len(p.Publics) < 1 {
		return fmt.Errorf("subprotocol started with an invlid public key list")
	} else if p.SubleaderTimeout < 1 {
		p.SubleaderTimeout = time.Duration(float64(DefaultProtocolTimeout) * subleaderTimeoutProportion)
	}
	announcement := StructAnnouncement{p.TreeNode(),
		Announcement{p.Proposal, p.Publics}}
	p.ChannelAnnouncement <- announcement
	return nil
}
