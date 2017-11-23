package service

import (
	"testing"

	"github.com/dedis/student_17_bftcosi"
	"github.com/stretchr/testify/assert"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
)

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestService_ClockRequest(t *testing.T) {
	log.SetDebugVisible(3) //TODO: remove once debugged
	local := onet.NewTCPTest()
	// generate 5 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	hosts, roster, _ := local.GenTree(5, true)
	defer local.CloseAll()

	services := local.GetServices(hosts, cosiID)

	for _, s := range services {
		log.Lvl2("Sending request to", s)
		resp, err := s.(*Service).ClockRequest(
			&cosi.ClockRequest{Roster: roster},
		)
		log.ErrFatal(err)

		//TODO: update
		log.Lvl2(resp)
		// assert.Equal(t, resp.Signature, len(roster.List))
	}
}

func TestService_CountRequest(t *testing.T) {
	local := onet.NewTCPTest()
	// generate 5 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	hosts, roster, _ := local.GenTree(5, true)
	defer local.CloseAll()

	services := local.GetServices(hosts, cosiID)

	for _, s := range services {
		log.Lvl2("Sending request to", s)
		resp, err := s.(*Service).ClockRequest(
			&cosi.ClockRequest{Roster: roster},
		)
		log.ErrFatal(err)
		assert.Equal(t, resp.Signature, len(roster.List))
		count, err := s.(*Service).CountRequest(&cosi.CountRequest{})
		log.ErrFatal(err)
		assert.Equal(t, 1, count.Count)
	}
}
