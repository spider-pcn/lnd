package htlcswitch

import (
	"time"
	"github.com/lightningnetwork/lnd/lnwire"
)

// This is the node class for the priority queue
// Note that the underlying data structure used is a min heap
type node struct {
	priority time.Time
  //priority lnwire.MilliSatoshi
	packet   *htlcPacket
}

// The priority is set by the time the packet arrives. This effectively makes
// it a fifo queue. We can modify this function to change the priority as we
// like.
func makeNode(pkt *htlcPacket) node {
	//p := time.Now()
	if htlc, ok := pkt.htlc.(*lnwire.UpdateAddHTLC); ok {
		deadline := htlc.Crafted.Add(htlc.Timeout)
		return node{
			priority: deadline,
			packet:   pkt,
		}
	}

	return node {
		priority: time.Now(),
		packet:   pkt,
	}

	//return node{
		//priority: pkt.amount,
		//packet:   pkt,
	//}
}

// priorityQueue type implements the heap.Interface
// To be used with heap module
type priorityQueue []node

// sort.Interface Less function
func (p priorityQueue) Less(i, j int) bool {
	return p[i].priority.After(p[j].priority)
    //return p[i].priority < p[j].priority
}

// sort.Interface Len function
func (p priorityQueue) Len() int {
	return len(p)
}

// sort.Interface Swap function
func (p priorityQueue) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

// heap.Interface Push function
func (p *priorityQueue) Push(x interface{}) {
	*p = append(*p, x.(node))
}

// heap.Interface Pop function
func (p *priorityQueue) Pop() interface{} {
	t := *p
	ret := t[len(t)-1]
	*p = t[:len(t)-1]
	return ret
}
