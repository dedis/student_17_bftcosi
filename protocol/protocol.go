package protocol

import (
	"fmt"
	"time"

	"github.com/dedis/student_17_bftcosi/cosi"
	"gopkg.in/dedis/crypto.v0/abstract"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/network"
	"gopkg.in/dedis/onet.v1/log"
)

//TODO: find and destroy "Node [...] already gone" warning message

//init() is done at startup. It defines every messages that is handled by the network
// and registers the protocols.
func init() {
	network.RegisterMessages(Announcement{}, Commitment{}, Challenge{}, Response{}, Stop{})

	onet.GlobalProtocolRegister(ProtocolName, NewProtocol)
	onet.GlobalProtocolRegister(subProtocolName, NewSubProtocol)
}
// CosiRootNode holds the parameters of the protocol.
// It also defines a channel that will receive the final signature.
type CosiRootNode struct {
	*onet.TreeNodeInstance
	Publics					[]abstract.Point

	NSubtrees      			int
	Proposal       			[]byte
	CreateProtocol 			CreateProtocolFunction
	ProtocolTimeout			time.Duration

	start					chan bool
	FinalSignature			chan []byte
}

type CreateProtocolFunction func(name string, t *onet.Tree) (onet.ProtocolInstance, error)

// The `NewProtocol` method is used to define the protocol.
func NewProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {

	var list []abstract.Point
	for _, t := range n.Tree().List() {
		list = append(list, t.ServerIdentity.Public)
	}

	c := &CosiRootNode{
		TreeNodeInstance:       n,
		Publics:				list,
		start: 					make(chan bool),
		FinalSignature:			make(chan []byte),
	}

	return c, nil
}

//Dispatch() is the main method of the protocol, defining the root node behaviour
// and sequential handling of subprotocols.
func (p *CosiRootNode) Dispatch() error {
	defer p.Done()

	if !p.IsRoot() {
		return nil
	}

	//generate trees
	nNodes := p.Tree().Size()
	trees, err := GenTrees(p.Tree().Roster, nNodes, p.NSubtrees)
	if err != nil {
		return fmt.Errorf("error in tree generation: %s", err)
	}

	//if one node, sign without subprotocols
	if nNodes == 1 {
		trees = make([]*onet.Tree, 0)
	}

	//wait for start signal
	<- p.start

	//start all subprotocols
	cosiProtocols := make([]*CosiSubProtocolNode, len(trees))
	for i, tree := range trees {
		cosiProtocols[i], err = p.startSubProtocol(tree)
		if err != nil {
			return err
		}
	}
	log.Lvl3("all protocols started")

	//get all commitments, restart subprotocols where subleaders do not respond
	commitments := make([]StructCommitment, len(trees))
	for i, cosiProtocol := range cosiProtocols {
		protocol := cosiProtocol
		for commitments[i].CosiCommitment == nil {
			select {
			case _ = <-protocol.subleaderNotResponding:
				log.Lvlf2("subleader from tree %d failed, restarting it", i)

				//send stop signal
				protocol.HandleStop(StructStop{protocol.TreeNode(), Stop{}})

				//generate new tree
				subleaderID := trees[i].Root.Children[0].RosterIndex
				newSubleaderID := subleaderID +1
				if newSubleaderID >= len(trees[i].Roster.List) {
					newSubleaderID = 1
				}
				trees[i], err = genSubtree(trees[i].Roster, newSubleaderID)
				if err != nil {
					return err
				}

				//restart protocol
				protocol, err = p.startSubProtocol(trees[i])
				if err != nil {
					return fmt.Errorf("error in restarting of protocol: %s", err)
				}
				cosiProtocols[i] = protocol
			case commitment := <-protocol.subCommitment:
				commitments[i] = commitment
			case <-time.After(p.ProtocolTimeout):
				return fmt.Errorf("didn't get commitment in time")
			}
		}
	}

	//generate challenge
	log.Lvl3("root-node generating global challenge")
	secret, commitment, mask, err := generatePersonnalCommitment(p.TreeNodeInstance, p.Publics, commitments)
	if err != nil {
		return err
	}
	cosiChallenge, err := cosi.Challenge(network.Suite, commitment,
		p.Root().PublicAggregateSubTree, p.Proposal)
	if err != nil {
		return err
	}
	structChallenge := StructChallenge{p.TreeNode(), Challenge{cosiChallenge}}

	//send challenge to every subprotocol
	for _, cosiProtocol := range cosiProtocols {
		protocol := cosiProtocol
		protocol.ChannelChallenge <- structChallenge
	}

	//get response from all subprotocols
	responses := make([]StructResponse, len(trees))
	for i, cosiProtocol := range cosiProtocols {
		protocol := cosiProtocol
		for responses[i].CosiReponse == nil {
			select {
			case response := <-protocol.subResponse:
				responses[i] = response
			case <-time.After(p.ProtocolTimeout):
				return fmt.Errorf("didn't finish in time")
			}
		}
	}

	//signs the proposal
	 response, err := generateResponse(p.TreeNodeInstance, responses, secret, cosiChallenge)
	if err != nil {
		return err
	}
	log.Lvl3(p.ServerIdentity().Address, "starts final signature")
	var signature []byte
	signature, err = cosi.Sign(p.Suite(), commitment, response, mask)
	if err != nil {
		return err
	}
	p.FinalSignature <- signature

	log.Lvl3("Root-node is done")
	return nil
}

// Start is done only by root and starts the protocol.
// It also verifies that the protocol has been correctly parameterized.
func (p *CosiRootNode) Start() error {
	if p.Proposal == nil {
		return fmt.Errorf("no proposal specified")
	} else if p.CreateProtocol == nil {
		return fmt.Errorf("no start function specified")
	} else if p.NSubtrees < 0 {
		p.NSubtrees = 1
	} else if p.ProtocolTimeout < 10 {
		p.ProtocolTimeout = DefaultProtocolTimeout
	}
	log.Lvl3("Starting Cosi")
	p.start <- true
	return nil
}

// startSubProtocol creates, parametrize and starts a subprotocol on a given tree
// and returns the started protocol.
func (p *CosiRootNode) startSubProtocol (tree *onet.Tree) (*CosiSubProtocolNode, error) {

	pi, err := p.CreateProtocol(subProtocolName, tree)
	if err != nil {
		return nil, err
	}

	cosiProtocol := pi.(*CosiSubProtocolNode)
	cosiProtocol.Publics = p.Publics
	cosiProtocol.Proposal = p.Proposal
	cosiProtocol.SubleaderTimeout = time.Duration(float64(p.ProtocolTimeout) * subleaderTimeoutProportion)

	err = cosiProtocol.Start()
	if err != nil {
		return nil, err
	}

	return cosiProtocol, err
}