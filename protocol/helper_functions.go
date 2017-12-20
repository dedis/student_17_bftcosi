package protocol

import (
	"github.com/dedis/student_17_bftcosi/cosi"
	"gopkg.in/dedis/crypto.v0/abstract"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
	"fmt"
	"gopkg.in/dedis/onet.v1/network"
)

// generateCommitmentAndAggregate generates a personal secret and commitment
// and returns respectively the secret, an aggregated commitment and an aggregated mask
func generateCommitmentAndAggregate(t *onet.TreeNodeInstance, publics []abstract.Point, structCommitments []StructCommitment) (abstract.Scalar, abstract.Point, *cosi.Mask, error) {

	if t == nil {
		return nil, nil, nil, fmt.Errorf("TreeNodeInstance should not be nil, but is")
	} else if publics == nil {
		return nil, nil, nil, fmt.Errorf("publics should not be nil, but is")
	} else if structCommitments == nil {
		return nil, nil, nil, fmt.Errorf("structCommitments should not be nil, but is")
	}

	//extract lists of commitments and masks
	var commitments []abstract.Point
	var masks [][]byte
	for _, c := range structCommitments {
		commitments = append(commitments, c.CoSiCommitment)
		masks = append(masks, c.Mask)
	}

	//generate personal secret and commitment
	secret, commitment := cosi.Commit(t.Suite(), nil)
	commitments = append(commitments, commitment)

	//generate personal mask
	personalMask, err := cosi.NewMask(t.Suite(), publics, t.Public())
	if err != nil {
		return nil, nil, nil, err
	}
	masks = append(masks, personalMask.Mask())

	//aggregate commitments and masks
	aggCommitment, aggMask, err :=
		cosi.AggregateCommitments(t.Suite(), commitments, masks)
	if err != nil {
		return nil, nil, nil, err
	}

	//create final aggregated mask
	finalMask, err := cosi.NewMask(t.Suite(), publics, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	err = finalMask.SetMask(aggMask)
	if err != nil {
		return nil, nil, nil, err
	}

	return secret, aggCommitment, finalMask, nil
}

// generateResponse generates a personal response based on the secret
// and returns the aggregated response of all children and the node
func generateResponse(t *onet.TreeNodeInstance, structResponses []StructResponse, secret abstract.Scalar, challenge abstract.Scalar) (abstract.Scalar, error) {

	if t == nil {
		return nil, fmt.Errorf("TreeNodeInstance should not be nil, but is")
	} else if structResponses == nil {
		return nil, fmt.Errorf("StructResponse should not be nil, but is")
	} else if secret == nil {
		return nil, fmt.Errorf("secret should not be nil, but is")
	} else if challenge == nil {
		return nil, fmt.Errorf("challenge should not be nil, but is")
	}

	//extract lists of responses
	var responses []abstract.Scalar
	for _, c := range structResponses {
		responses = append(responses, c.CoSiReponse)
	}

	//generate personal response
	personalResponse, err := cosi.Response(t.Suite(), t.Private(), secret, challenge)
	if err != nil {
		return nil, err
	}
	responses = append(responses, personalResponse)

	//aggregate responses
	aggResponse, err := cosi.AggregateResponses(t.Suite(), responses)
	if err != nil {
		return nil, err
	}

	log.Lvl3(t.ServerIdentity().Address, "is done aggregating responses with total of",
		len(responses), "responses")

	return aggResponse, nil
}


func GetSubleaderIDs(tree *onet.Tree, nNodes, nSubtrees int) ([]network.ServerIdentityID, error) {
	exampleTrees, err := GenTrees(tree.Roster, nNodes, nSubtrees)
	if err != nil {
		return nil, fmt.Errorf("error in creation of example tree:%s", err)
	}
	subleadersIDs := make([]network.ServerIdentityID, 0)
	for _, subtree := range exampleTrees {
		if len(subtree.Root.Children) < 1 {
			return nil, fmt.Errorf("expected a subtree with at least a subleader, but found none")
		}
		subleadersIDs = append(subleadersIDs, subtree.Root.Children[0].ServerIdentity.ID)
	}
	return subleadersIDs, nil
}


func GetLeafsIDs(tree *onet.Tree, nNodes, nSubtrees int) ([]network.ServerIdentityID, error) {
	exampleTrees, err := GenTrees(tree.Roster, nNodes, nSubtrees)
	if err != nil {
		return nil, fmt.Errorf("error in creation of example tree:%s", err)
	}
	leafsIDs := make([]network.ServerIdentityID, 0)
	for _, subtree := range exampleTrees {
		if len(subtree.Root.Children) < 1 {
			return nil, fmt.Errorf("expected a subtree with at least a subleader, but found none")
		}
		for _, leaf := range subtree.Root.Children[0].Children {
			leafsIDs = append(leafsIDs, leaf.ServerIdentity.ID)
		}
	}
	return leafsIDs, nil
}