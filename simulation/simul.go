package main

import (
	// Service needs to be imported here to be instantiated.
	_ "github.com/dedis/student_17_bftcosi/service"
	"gopkg.in/dedis/onet.v1/simul"
)

func main() {
	simul.Start()
}
