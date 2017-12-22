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

//init() is done at startup. It defines every messages that is handled by the network
// and registers the protocols.
func init() {
	network.RegisterMessages(Announcement{}, Commitment{}, Challenge{}, Response{}, Stop{})

	onet.GlobalProtocolRegister(ProtocolName, NewProtocol)
	onet.GlobalProtocolRegister(subProtocolName, NewSubProtocol)
}
// CoSiRootNode holds the parameters of the protocol.
// It also defines a channel that will receive the final signature.
type CoSiRootNode struct {
	*onet.TreeNodeInstance
	Publics					[]abstract.Point

	NSubtrees      			int
	Proposal       			[]byte
	CreateProtocol 			CreateProtocolFunction
	ProtocolTimeout			time.Duration
	SubleaderTimeout		time.Duration
	LeavesTimeout			time.Duration
	hasStopped       		bool //used since Shutdown can be called multiple time

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

	c := &CoSiRootNode{
		TreeNodeInstance:       n,
		Publics:				list,
		hasStopped:				false,
		start: 					make(chan bool),
		FinalSignature:			make(chan []byte),
	}

	return c, nil
}


func (p *CoSiRootNode) Shutdown() error {
	if !p.hasStopped {
		close(p.start)
		p.hasStopped = true
	}
	return nil
}

//Dispatch() is the main method of the protocol, defining the root node behaviour
// and sequential handling of subprotocols.
func (p *CoSiRootNode) Dispatch() error {

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
	_, channelOpen := <- p.start
	if !channelOpen {
		return nil
	}

	//start all subprotocols
	coSiSubProtocols := make([]*CoSiSubProtocolNode, len(trees))
	for i, tree := range trees {
		coSiSubProtocols[i], err = p.startSubProtocol(tree)
		if err != nil {
			return err
		}
	}
	log.Lvl3("all protocols started")

	//get all commitments, restart subprotocols where subleaders do not respond
	commitments := make([]StructCommitment, 0)
	runningSubProtocols := make([]*CoSiSubProtocolNode, 0)
	subtrees:
	for i, subProtocol := range coSiSubProtocols {
		for {
			select {
			case _ = <-subProtocol.subleaderNotResponding:
				log.Lvlf2("subleader from tree %d failed, restarting it", i)

				//send stop signal
				subProtocol.HandleStop(StructStop{subProtocol.TreeNode(), Stop{}})

				//generate new tree
				subleaderID := trees[i].Root.Children[0].RosterIndex
				newSubleaderID := subleaderID +1
				if newSubleaderID >= len(trees[i].Roster.List) {
					log.Lvl2("subprotocol", i,  "failed with every subleader, ignoring this subtree")
					continue subtrees
				}
				trees[i], err = GenSubtree(trees[i].Roster, newSubleaderID)
				if err != nil {
					return err
				}

				//restart subprotocol
				subProtocol, err = p.startSubProtocol(trees[i])
				if err != nil {
					return fmt.Errorf("error in restarting of subprotocol: %s", err)
				}
			case commitment := <-subProtocol.subCommitment:
				runningSubProtocols = append(runningSubProtocols,subProtocol)
				commitments = append(commitments, commitment)
				continue subtrees
			case <-time.After(p.ProtocolTimeout):
				return fmt.Errorf("didn't get commitment in time")
			}
		}
	}

	//generate challenge
	log.Lvl3("root-node generating global challenge")
	secret, commitment, finalMask, err := generateCommitmentAndAggregate(p.TreeNodeInstance, p.Publics, commitments)
	if err != nil {
		return err
	}

	coSiChallenge, err := cosi.Challenge(p.Suite(), commitment, finalMask.AggregatePublic, p.Proposal)
	if err != nil {
		return err
	}
	structChallenge := StructChallenge{p.TreeNode(), Challenge{coSiChallenge}}

	//send challenge to every subprotocol
	for _, coSiProtocol := range runningSubProtocols {
		subProtocol := coSiProtocol
		subProtocol.ChannelChallenge <- structChallenge
	}

	//get response from all subprotocols
	responses := make([]StructResponse, 0)
	for _, cosiSubProtocol := range runningSubProtocols {
		subProtocol := cosiSubProtocol
		select {
		case response := <-subProtocol.subResponse:
			responses = append(responses, response)
			continue
		case <-time.After(p.ProtocolTimeout):
			return fmt.Errorf("didn't finish in time")
		}
	}

	//signs the proposal
	 response, err := generateResponse(p.TreeNodeInstance, responses, secret, coSiChallenge)
	if err != nil {
		return err
	}
	log.Lvl3(p.ServerIdentity().Address, "starts final signature")
	var signature []byte
	signature, err = cosi.Sign(p.Suite(), commitment, response, finalMask)
	if err != nil {
		return err
	}
	p.FinalSignature <- signature

	log.Lvl3("Root-node is done without errors")
	return nil
}

// Start is done only by root and starts the protocol.
// It also verifies that the protocol has been correctly parameterized.
func (p *CoSiRootNode) Start() error {
	if p.Proposal == nil {
		return fmt.Errorf("no proposal specified")
	} else if p.CreateProtocol == nil {
		return fmt.Errorf("no create protocol function specified")
	} else if p.NSubtrees < 1 {
		p.NSubtrees = 1
	}
	if p.ProtocolTimeout < 10 {
		p.ProtocolTimeout = DefaultProtocolTimeout
	}
	if p.SubleaderTimeout < 10 {
		p.SubleaderTimeout = DefaultSubleaderTimeout
	}
	if p.LeavesTimeout < 10 {
		p.LeavesTimeout = DefaultLeavesTimeout
	}

	log.Lvl3("Starting CoSi")
	p.start <- true
	return nil
}

// startSubProtocol creates, parametrize and starts a subprotocol on a given tree
// and returns the started protocol.
func (p *CoSiRootNode) startSubProtocol (tree *onet.Tree) (*CoSiSubProtocolNode, error) {

	pi, err := p.CreateProtocol(subProtocolName, tree)
	if err != nil {
		return nil, err
	}

	coSiSubProtocol := pi.(*CoSiSubProtocolNode)
	coSiSubProtocol.Publics = p.Publics
	coSiSubProtocol.Proposal = p.Proposal
	coSiSubProtocol.SubleaderTimeout = p.SubleaderTimeout
	coSiSubProtocol.LeavesTimeout = p.LeavesTimeout

	err = coSiSubProtocol.Start()
	if err != nil {
		return nil, err
	}

	return coSiSubProtocol, err
}