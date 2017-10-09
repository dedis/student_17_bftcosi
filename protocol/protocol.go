package protocol

/*
The `NewProtocol` method is used to define the protocol and to register
the handlers that will be called if a certain type of message is received.
The handlers will be treated according to their signature.

The protocol-file defines the actions that the protocol needs to do in each
step. The root-node will call the `Start`-method of the protocol. Each
node will only use the `Handle`-methods, and not call `Start` again.
*/

import (
	"errors"

	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
	"github.com/dedis/student_17_bftcosi/cosi"
	"gopkg.in/dedis/crypto.v0/abstract"
)

func init() {
	network.RegisterMessage(Announcement{})
	network.RegisterMessage(Commitment{})
	network.RegisterMessage(Challenge{})
	network.RegisterMessage(Response{})

	onet.GlobalProtocolRegister(Name, NewProtocol)
}


// Cosi just holds a message that is passed to all children. It
// also defines a channel that will receive the final aggregate. Only the
// root-node will write to the channel.
type Cosi struct {
	*onet.TreeNodeInstance
	List                []abstract.Point
	MinShardSize        int // can be one more
	Seed                int
	Proposal            []byte
	AggregateCommitment chan Commitment
}

// NewProtocol initialises the structure for use in one round
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
		TreeNodeInstance:    n,
		List:                list,
		MinShardSize:        n.Tree().Size()-1 / nShards,
		Seed:                13213, //TODO: see how generate
		Proposal:            make([]byte, 0),
		AggregateCommitment: make(chan Commitment),
	} //TODO: see if should add TreeNodeIndex

	for _, handler := range []interface{}{c.HandleAnnouncement, c.HandleCommitment} {
		err := c.RegisterHandler(handler)
		if err != nil {
			return nil, errors.New("couldn't register handler: " + err.Error())
		}
	}
	return c, nil
}

// Start sends the Announcement-message to all children
func (p *Cosi) Start() error {
	log.Lvl3("Starting Cosi")
	return p.HandleAnnouncement(StructAnnouncement{p.TreeNode(),
		Announcement{p.MinShardSize, p.Seed, p.Proposal}})
}

// HandleAnnouncement announce the start of the protocol by the leader (tree root) to all nodes.
func (p *Cosi) HandleAnnouncement(msg StructAnnouncement) error {
	if !p.IsLeaf() {
		// If we have children, send the same message to all of them
		log.Lvl3(p.ServerIdentity().Address, "is sending announcement to children(s)")
		p.SendToChildren(&msg.Announcement)
	} else {
		// If we're the leaf, start to reply
		log.Lvl3(p.ServerIdentity().Address, "begins commitment")
		p.HandleCommitment([]StructCommitment{})
	}
	return nil
}

// HandleReply is the message going up the tree and holding a counter
// to verify the number of nodes.
func (p *Cosi) HandleCommitment(structCommitments []StructCommitment) error {
	defer p.Done() //TODO: move this instruction to final state once implemented
	log.Lvl3(p.ServerIdentity().Address, ": received commitment to handle")

	//extract lists of commitments and masks
	var commitments []abstract.Point
	var masks [][]byte
	for _, c := range structCommitments {
		commitments = append(commitments, c.CosiCommitment)
		masks = append(masks, c.Mask.Mask())
	}

	//generate personal commitment
	//TODO: grab first argument for reuse in response
	_, commitment := cosi.Commit(p.Suite(), nil) //TODO: check if should use a given stream instead of random one
	commitments = append(commitments, commitment)

	//generate personal mask
	//mask, err := cosi.NewMask(p.Suite(), p.List, p.TreeNode().PublicAggregateSubTree)
	//if err != nil {
	//	return err
	//}
	//masks = append(masks, mask.Mask())
	mask := make([]byte, 0)
	masks = append(masks, mask)
	var err error

	//aggregate commitments and masks
	var aggCommitment Commitment
	//aggCommitment.Mask = *mask
	var aggMask []byte
	aggCommitment.CosiCommitment, aggMask, err =
		cosi.AggregateCommitments(p.Suite(), commitments, masks)
	if err != nil {
		return err
	}
	err = aggCommitment.Mask.SetMask(aggMask)
	if err != nil {
		return err
	}

	log.Lvl3(p.ServerIdentity().Address, "is done aggregating commitments with total of",
		len(commitments), "commitments")
	if !p.IsRoot() {
		log.Lvl3(p.ServerIdentity().Address, ": Sending to parent")
		return p.SendToParent(&aggCommitment)
	}
	log.Lvl3("Root-node is done")
	p.AggregateCommitment <- aggCommitment
	return nil
}
