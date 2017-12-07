package protocol

import (
	"github.com/dedis/student_17_bftcosi/cosi"
	"gopkg.in/dedis/crypto.v0/abstract"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
)

// generatePersonalCommitment generates a personal secret and commitment
// and returns respectively the secret, an aggregated commitment and an aggregated mask
func generatePersonalCommitment(t *onet.TreeNodeInstance, publics []abstract.Point, structCommitments []StructCommitment) (abstract.Scalar, abstract.Point, *cosi.Mask, error) {

	//extract lists of commitments and masks
	var commitments []abstract.Point
	var masks [][]byte
	for _, c := range structCommitments {
		commitments = append(commitments, c.CosiCommitment)
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
	finalMask.SetMask(aggMask)

	return secret, aggCommitment, finalMask, nil
}

// generateResponse generates a personal response based on the secret
// and returns the aggregated response of all children and the node
func generateResponse(t *onet.TreeNodeInstance, structResponse []StructResponse, secret abstract.Scalar, challenge abstract.Scalar) (abstract.Scalar, error) {

	//extract lists of responses
	var responses []abstract.Scalar
	for _, c := range structResponse {
		responses = append(responses, c.CosiReponse)
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
