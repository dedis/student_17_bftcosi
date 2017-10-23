/*
Package protocol contains an implementation of the Cosi protocol as described in the paper
"Keeping Authorities "Honest or Bust" with Decentralized Witness Cosigning"

The protocol has four messages:
	- Announcement which is sent from the root down the tree and announce the proposal
	- Commitment which is sent back up to the root, containing an aggregated commitment from all nodes
	- Challenge which is sent from the root down the tree and contains the aggregated challenge
	- Response which is sent back up to the root, containing the final aggregated signature, then used by the root to sign the proposal

The protocol uses four files:
- struct.go defines the messages sent around
- protocol.go defines the actions for each message
- gen_tree.go contains the function that generates the basic tree
- simulation.go tests the protocol on distant platforms like deterlab //TODO R: implement

The package protocol_tests contains unit tests testing the package's code.
*/
package protocol
