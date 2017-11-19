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

func init() {
	network.RegisterMessage(Announcement{})
	network.RegisterMessage(Commitment{})
	network.RegisterMessage(Challenge{})
	network.RegisterMessage(Response{})
	//TODO: define a stop message

	onet.GlobalProtocolRegister(ProtocolName, NewProtocol)
	onet.GlobalProtocolRegister(SubProtocolName, NewSubProtocol)
}

type CosiRootNode struct {
	*onet.TreeNodeInstance
	Publics					[]abstract.Point

	NSubtrees      			int
	Proposal       			[]byte
	CreateProtocol 			CreateProtocolFunction

	start					chan bool
	FinalSignature			chan []byte
}

type CreateProtocolFunction func(name string, t *onet.Tree) (onet.ProtocolInstance, error)

// The `NewProtocol` method is used to define the protocol and to register
// the channels where the messages will be received.
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

//Dispatch() is the main method of the protocol, handling the rot node behaviour
func (p *CosiRootNode) Dispatch() error {
	defer p.Done()

	if !p.IsRoot() {
		return nil
	}

	//generate trees
	nNodes := p.Tree().Size()
	trees, err := GenTrees(p.Tree().Roster, nNodes, p.NSubtrees)
	if err != nil {
		return fmt.Errorf("Error in tree generation:", err)
	}

	//if one node, do the signature without subprotocols
	if nNodes == 1 {
		trees = make([]*onet.Tree, 0)
	}

	//wait for start signal
	<- p.start

	//start all subprotocols
	cosiProtocols := make([]*CosiSubProtocolNode, len(trees))
	for i, tree := range trees {
		//start protocol
		pi, err := p.CreateProtocol(SubProtocolName, tree)
		if err != nil {
			return err
		}
		cosiProtocols[i] = pi.(*CosiSubProtocolNode)
		cosiProtocols[i].Publics = p.Publics
		cosiProtocols[i].Proposal = p.Proposal
		cosiProtocols[i].Start()
	}
	log.Lvl3("all protocols started")

	//get all commitments, restart subprotocols where subleaders do not respond
	commitments := make([]StructCommitment, len(trees))
	timeout := 4*Timeout
	for i, cosiProtocol := range cosiProtocols {
		protocol := cosiProtocol
		for commitments[i].CosiCommitment == nil {
			select {
			case _ = <-protocol.subleaderNotResponding:
				log.Lvl3("subleader %d failed, restarting it", i)
				//restart protocol
				pi, err := p.CreateProtocol(ProtocolName, trees[i])
				if err != nil {
					return err
				}
				protocol = pi.(*CosiSubProtocolNode)
			case commitment := <-protocol.subCommitment:
				commitments[i] = commitment
			case <-time.After(timeout):
				return fmt.Errorf("Didn't get commitment in time")
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
			case <-time.After(timeout):
				return fmt.Errorf("Didn't finish in time")
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

// Start is done only by root and starts the protocol
func (p *CosiRootNode) Start() error {
	log.Lvl3("Starting Cosi")

	if p.Proposal == nil {
		return fmt.Errorf("No proposal specified")
	} else if p.CreateProtocol == nil {
		return fmt.Errorf("No start function specified")
	} else if p.NSubtrees < 0 {
		p.NSubtrees = 1
	}
	p.start <- true
	return nil
}