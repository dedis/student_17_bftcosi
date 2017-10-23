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
// also defines a channel that will receive the final signature. Only the
// root-node will write to the channel.

type Cosi struct {
	*onet.TreeNodeInstance
	List                []abstract.Point
	MinShardSize        int // can be one more
	Seed                int
	Proposal            []byte
	secret              abstract.Scalar
	aggregateMask       *cosi.Mask
	aggregateCommitment abstract.Point
	Challenge           abstract.Scalar
	FinalSignature      chan []byte
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
		FinalSignature:		make(chan []byte),
	}

	for _, handler := range []interface{}{c.HandleAnnouncement, c.HandleCommitment, c.HandleChallenge, c.HandleResponse} {
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
	p.Seed = 13213
	p.Proposal = []byte{0xFF}
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

//TODO: handle timeout in the mask

// HandleCommitment is the message going up the tree
func (p *Cosi) HandleCommitment(structCommitments []StructCommitment) error {
	log.Lvl3(p.ServerIdentity().Address, ": received commitment to handle")

	//extract lists of commitments and masks
	var commitments []abstract.Point
	var masks [][]byte
	for _, c := range structCommitments {
		commitments = append(commitments, c.CosiCommitment)
		masks = append(masks, c.Mask)
	}

	//generate personal commitment
	var commitment abstract.Point
	p.secret, commitment = cosi.Commit(p.Suite(), nil)
	commitments = append(commitments, commitment)

	//generate personal mask
	var err error
	p.aggregateMask, err = cosi.NewMask(p.Suite(), p.List, p.TreeNode().PublicAggregateSubTree)
	if err != nil {
		return err
	}
	masks = append(masks, p.aggregateMask.Mask())

	//aggregate commitments and masks
	var aggMask []byte
	p.aggregateCommitment, aggMask, err =
		cosi.AggregateCommitments(p.Suite(), commitments, masks)
	if err != nil {
		return err
	}
	p.aggregateMask.SetMask(aggMask)

	log.Lvl3(p.ServerIdentity().Address, "is done aggregating commitments with total of",
		len(commitments), "commitments")


	if !p.IsRoot() {
		log.Lvl3(p.ServerIdentity().Address, ": Sending to parent")
		return p.SendToParent(&Commitment{p.aggregateCommitment,p.aggregateMask.Mask()})
	}

	//if root, generate challenge
	log.Lvl3("Root-node is done aggregating commitments")
	challenge, err := cosi.Challenge(p.Suite(), p.aggregateCommitment,
		p.Root().PublicAggregateSubTree, p.Proposal)
	if err != nil {
		return err
	}
	return p.HandleChallenge(StructChallenge{p.TreeNode(), Challenge{challenge}})
}

// HandleChallenge propagates the cosi challenge to all nodes.
func (p *Cosi) HandleChallenge(msg StructChallenge) error {
	p.Challenge = msg.CosiChallenge
	if !p.IsLeaf() {
		// If we have children, send the same message to all of them
		log.Lvl3(p.ServerIdentity().Address, "is sending challenge to children(s)")
		p.SendToChildren(&msg.Challenge)
	} else {
		// If we're the leaf, start to reply
		log.Lvl3(p.ServerIdentity().Address, "begins response")
		p.HandleResponse([]StructResponse{})
	}
	return nil
}

// HandleResponse returns the aggregated response of all children and the node up the tree
func (p *Cosi) HandleResponse(structResponse []StructResponse) error {
	defer p.Done()
	log.Lvl3(p.ServerIdentity().Address, ": received Response to handle")

	//extract lists of responses
	var responses []abstract.Scalar
	for _, c := range structResponse {
		responses = append(responses, c.CosiReponse)
	}

	//generate personal response
	response, err := cosi.Response(p.Suite(), p.TreeNodeInstance.Private(), p.secret, p.Challenge)
	if err != nil {
		return err
	}
	responses = append(responses, response)

	//aggregate responses
	var aggResponse Response
	aggResponse.CosiReponse, err = cosi.AggregateResponses(p.Suite(), responses)
	if err != nil {
		return err
	}

	log.Lvl3(p.ServerIdentity().Address, "is done aggregating responses with total of",
		len(responses), "responses")

	if !p.IsRoot() {
		log.Lvl3(p.ServerIdentity().Address, ": Sending to parent")
		return p.SendToParent(&aggResponse)
	}

	//if node is root
	log.Lvl3("Root-node is done")
	var signature []byte
	signature, err = cosi.Sign(p.Suite(), p.aggregateCommitment, aggResponse.CosiReponse, p.aggregateMask)
	if err != nil {
		return err
	}
	p.FinalSignature <- signature
	return nil
}