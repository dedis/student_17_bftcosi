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
	list []*onet.TreeNode
	shardSize int
	seed int
	proposal []byte
	AggregateCommitment chan Commitment
}

// NewProtocol initialises the structure for use in one round
func NewProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	c := &Cosi{
		TreeNodeInstance: n,
		list: n.Tree().List(),
		shardSize: len(n.Root().Children),
		seed: 13213, //TODO: see how generate
		proposal: nil,
		AggregateCommitment:       make(chan Commitment),
	} //TODO: see if should add TreeNodeIndex

	for _, handler := range []interface{}{c.HandleAnnouncement, c.HandleCommitment} {
		if err := c.RegisterHandler(handler); err != nil {
			return nil, errors.New("couldn't register handler: " + err.Error())
		}
	}
	return c, nil
}

// Start sends the Announcement-message to all children
func (p *Cosi) Start() error {
	log.Lvl3("Starting Cosi")
	return p.HandleAnnouncement(StructAnnouncement{p.TreeNode(),
		Announcement{p.list, p.shardSize, p.seed, p.proposal}})
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
	defer p.Done() //TODO: remove once final state is implemented
	log.Lvl3(p.ServerIdentity().Address, ": received commitment to handle")

	var commitments []abstract.Point
	var masks [][]byte
	for _, c := range structCommitments {
		commitments = append(commitments, c.CosiCommitment)
		masks = append(masks, c.NodeData)
	}

	//generate personal commitment
	//TODO: grab first argument for reuse in response
	_, commitment := cosi.Commit(p.Suite(), nil) //TODO: check if should use a given stream instead of random one
	commitments = append(commitments, commitment)

	//generate personal mask
	err, mask := generateMask(p.list, p.TreeNode().ID)
	if err != nil {
		return err
	}
	masks = append(masks, mask)


	var aggCommitment Commitment
	aggCommitment.CosiCommitment, aggCommitment.NodeData, aggCommitment.Exception =
		cosi.AggregateCommitments(p.Suite(), commitments, masks)

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

//generate mask for one given node in O(n)
func generateMask(list []*onet.TreeNode, ID onet.TreeNodeID) (error, []byte) {

	mask := make([]byte, len(list))
	foundIndex := -1
	for i, n := range list {
		if n.ID == ID {
			foundIndex = i
		}
	}
	if foundIndex == -1 {
		return errors.New("node index not found in list"), nil
	}
	mask[foundIndex] = 0xFF
	return nil, mask
}