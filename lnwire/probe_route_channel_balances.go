package lnwire

import ()

// ProbeRouteChannelBalances is a message sent by a node in order to query the balance on a set of channels
// specified by the route on the probe message. The receiving node adds the balance on the channel between
// itself and the next hop on the route to the list of balances that are already contained within the message.
// The last node is responsible for sending this back to the sender
type ProbeRouteChannelBalances struct {
	// Route is the route for which we are probing the bandwidth
	// denoted by a slice of public keys of the nodes on the route
	Route []ChannelHop

	// HopNum denotes where in the probe we are and is helpful to directly
	// find the Next Hop
	HopNum int

	// routerChannelBalMap maps a router to the balance on the outgoing
	// channel to the next hop as specified by the router
	// this is filled as the probe message propagates
	RouterChannelBalMap map[Vertex]uint64

	// sender denotes the sender of the probe message; used to send the information
	// back to the sender after it is filled
	Sender Vertex

	// ProbeCompleted denotes whether or not all the information for all
	// channels along the path has been collected
	// if true the probe is on its way back to the sender
	ProbeCompleted bool
}

// NewProbeRouteChannelBalances creates a new empty ProbeRouteChannelBalances message
func NewProbeRouteChannelBalances() *ProbeRouteChannelBalances {
	return &ProbeRouteChannelBalances{}
}

// A compile time check to ensure ProbeRouteChannelBalances implements the
// lnwire.Message interface.
var _ Message = (*ProbeRouteChannelBalances)(nil)

// Decode deserializes a serialized ProbeRouteChannelBalances message stored in the
// passed io.Reader observing the specified protocol version.
//
// This is part of the lnwire.Message interface.
func (q *ProbeRouteChannelBalances) Decode(r io.Reader, pver uint32) error {
	return readElements(r,
		q.Route[:],
		q.RouterChannelBalMap,
		&q.Sender,
	)
}

// Encode serializes the target ProbeRouteChannelBalances into the passed io.Writer
// observing the protocol version specified.
//
// This is part of the lnwire.Message interface.
func (q *ProbeRouteChannelBalances) Encode(w io.Writer, pver uint32) error {
	return writeElements(w,
		q.Route[:],
		q.RouterChannelBalMap,
		q.Sender,
	)
}

// MsgType returns the integer uniquely identifying this message type on the
// wire.
//
// This is part of the lnwire.Message interface.
func (q *ProbeRouteChannelBalances) MsgType() MessageType {
	return MsgProbeRouteChannelBalances
}

// MaxPayloadLength returns the maximum allowed payload size for a
// ProbeRouteChannelBalances complete message observing the specified protocol version.
//
// This is part of the lnwire.Message interface.
func (q *ProbeRouteChannelBalances) MaxPayloadLength(uint32) uint32 {
	// 32 + 4 + 4
	// TODO: fix this vibhaa
	return 40
}

//TODO: define vertex
