package lnwire

import "io"

// FIXME: make this a common field / reuse the common parts of this struct type
// Vertex is a simple alias for the serialization of a compressed Bitcoin
// public key.
//type Vertex [33]byte

// ProbeRouteChannelPrices is a message sent by a node in order to query the balance on a set of channels
// specified by the route on the probe message. The receiving node adds the balance on the channel between
// itself and the next hop on the route to the list of balances that are already contained within the message.
// The last node is responsible for sending this back to the sender
type ProbeRouteChannelPrices struct {
	// Route is the route for which we are probing the bandwidth
	// denoted by a slice of public keys of the nodes on the route
	Route []Vertex

	// HopNum denotes where in the probe we are and is helpful to directly
	// find the Next Hop
	HopNum uint32

	// routerChannelBalMap maps a router to the balance on the outgoing
	// channel to the next hop as specified by the router
	// this is filled as the probe message propagates
	// RouterChannelBalances []MilliSatoshi
	RouterChannelPrices []MilliSatoshi

	// sender denotes the sender of the probe message; used to send the information
	// back to the sender after it is filled
	Sender Vertex

	// ProbeCompleted denotes whether or not all the information for all
	// channels along the path has been collected
	// if true the probe is on its way back to the sender
	ProbeCompleted uint8

	// CurrentNode denotes the current Node in the path that the probe is traversing
	// CurrentNode = Route[HopNum]
	CurrentNode Vertex

	// PathID denotes the path identifier as per the k-shortest paths - 0 denotes
	// the shortest path
	PathID uint32

	// Did probe result in some error
	Error uint8
}

// NewProbeRouteChannelPrices creates a new empty ProbeRouteChannelPrices message
func NewProbeRouteChannelPrices() *ProbeRouteChannelPrices {
	return &ProbeRouteChannelPrices{}
}

// A compile time check to ensure ProbeRouteChannelPrices implements the
// lnwire.Message interface.
var _ Message = (*ProbeRouteChannelPrices)(nil)

// Decode deserializes a serialized ProbeRouteChannelPrices message stored in the
// passed io.Reader observing the specified protocol version.
//
// This is part of the lnwire.Message interface.
func (q *ProbeRouteChannelPrices) Decode(r io.Reader, pver uint32) error {
	return readElements(r,
		&q.Route,
		&q.HopNum,
		&q.RouterChannelPrices,
		&q.Sender,
		&q.ProbeCompleted,
		&q.CurrentNode,
		&q.PathID,
		&q.Error,
	)
}

// Encode serializes the target ProbeRouteChannelPrices into the passed io.Writer
// observing the protocol version specified.
//
// This is part of the lnwire.Message interface.
func (q *ProbeRouteChannelPrices) Encode(w io.Writer, pver uint32) error {
	return writeElements(w,
		q.Route,
		q.HopNum,
		q.RouterChannelPrices,
		q.Sender,
		q.ProbeCompleted,
		q.CurrentNode,
		q.PathID,
		q.Error,
	)
}

// MsgType returns the integer uniquely identifying this message type on the
// wire
//
// This is part of the lnwire.Message interface.
func (q *ProbeRouteChannelPrices) MsgType() MessageType {
	return MsgProbeRouteChannelPrices
}

// MaxPayloadLength returns the maximum allowed payload size for a
// ProbeRouteChannelPrices complete message observing the specified protocol version.
//
// This is part of the lnwire.Message interface.
func (q *ProbeRouteChannelPrices) MaxPayloadLength(uint32) uint32 {
	// 32 + 4 + 4
	// TODO: fix this vibhaa
	return 65533
}

//TODO: define vertex
