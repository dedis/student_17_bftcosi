package cosi_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	// We need to include the service so it is started.
	"github.com/dedis/student_17_bftcosi"
	_ "github.com/dedis/student_17_bftcosi/service"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
)

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestClient_Clock(t *testing.T) {
	log.SetDebugVisible(3) //TODO: remove once debugged
	nbr := 5
	local := onet.NewTCPTest()
	// generate 5 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	_, roster, _ := local.GenTree(nbr, true)
	defer local.CloseAll()

	c := cosi.NewClient()
	cl1, err := c.Clock(roster)
	log.ErrFatal(err)
	assert.Equal(t, nbr, cl1.Signature) //TODO: update
	cl2, err := c.Clock(roster)
	log.ErrFatal(err)
	assert.Equal(t, nbr, cl2.Signature) //TODO: update
}

func TestClient_Count(t *testing.T) {
	nbr := 5
	local := onet.NewTCPTest()
	// generate 5 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	_, roster, _ := local.GenTree(nbr, true)
	defer local.CloseAll()

	c := cosi.NewClient()
	// Verify it's all 0s before
	for _, s := range roster.List {
		count, err := c.Count(s)
		log.ErrFatal(err)
		assert.Equal(t, 0, count)
	}

	// Make some clock-requests
	for range roster.List {
		_, err := c.Clock(roster)
		log.ErrFatal(err)
	}

	// Verify we have the correct total of requests
	total := 0
	for _, s := range roster.List {
		count, err := c.Count(s)
		log.ErrFatal(err)
		total += count
	}
	assert.Equal(t, nbr, total)
}
