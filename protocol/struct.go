package protocol

/*
Struct holds the messages that will be sent around in the protocol. You have
to define each message twice: once the actual message, and a second time
with the `*onet.TreeNode` embedded. The latter is used in the handler-function
so that it can find out who sent the message.
*/

import "gopkg.in/dedis/onet.v1"

// Name can be used from other packages to refer to this protocol.
const Name = "Template"

type Announcement struct {
	 list []*onet.TreeNode
	 shardSize int
	 seed int
}

// StructAnnouncement just contains Announcement and the data necessary to identify and
// process the message in the sda framework.
type StructAnnouncement struct {
	*onet.TreeNode
	Announcement
}

type Commitment struct {
	cosiCommitment []byte //uint64?
	nodeData []byte
	//exception if doesn't want to commit?
}

// StructCommitment just contains Commitment and the data necessary to identify and
// process the message in the sda framework.
type StructCommitment struct {
	*onet.TreeNode
	Commitment
}

type Challenge struct {
	cosiChallenge []byte //uint64?
	proposal []byte
}

// StructChallenge just contains Challenge and the data necessary to identify and
// process the message in the sda framework.
type StructChallenge struct {
	*onet.TreeNode
	Challenge
}

type Response struct {
	cosiReponse []byte //uint64?
	exception error //if the node doesn't want to sign
}

// StructResponse just contains Response and the data necessary to identify and
// process the message in the sda framework.
type StructResponse struct {
	*onet.TreeNode
	Response
}
