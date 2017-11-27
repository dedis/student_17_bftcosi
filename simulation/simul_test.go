package main_test

import (
	"testing"

	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/simul"
)

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestSimulation(t *testing.T) {
	//log.SetDebugVisible(3)
	simul.Start("protocol.toml")
}
