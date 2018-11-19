package htlcswitch

import (
	"container/heap"
	"sync"
	"sync/atomic"
	"time"
	"fmt"
	"github.com/lightningnetwork/lnd/lnwire"
)

// packetQueue is a goroutine-safe queue of htlc packets which over flow the
// current commitment transaction. An HTLC will overflow the current commitment
// transaction if one attempts to add a new HTLC to the state machine which
// already has the max number of pending HTLC's present on the commitment
// transaction.  Packets are removed from the queue by the channelLink itself
// as additional slots become available on the commitment transaction itself.
// In order to synchronize properly we use a semaphore to allow the channelLink
// to signal the number of slots available, and a condition variable to allow
// the packetQueue to know when new items have been added to the queue.
type packetQueue struct {
	// totalHtlcAmt is the sum of the value of all pending HTLC's currently
	// residing within the overflow queue. This value should only read or
	// modified *atomically*.
	totalHtlcAmt int64 // To be used atomically.

	// queueLen is an internal counter that reflects the size of the queue
	// at any given instance. This value is intended to be use atomically
	// as this value is used by internal methods to obtain the length of
	// the queue w/o grabbing the main lock. This allows callers to avoid a
	// deadlock situation where the main goroutine is attempting a send
	// with the lock held.
	queueLen int32 // To be used atomically.

	// maximum queue length for overflow queue
	maxQueueLen int32

	streamShutdown int32 // To be used atomically.

	queue priorityQueue

	wg sync.WaitGroup

	// freeSlots serves as a semaphore who's current value signals the
	// number of available slots on the commitment transaction.
	freeSlots chan struct{}

	queueCond *sync.Cond
	queueMtx  sync.Mutex

	// outgoingPkts is a channel that the channelLink will receive on in
	// order to drain the packetQueue as new slots become available on the
	// commitment transaction.
	outgoingPkts chan *htlcPacket

	quit chan struct{}
}

// newPacketQueue returns a new instance of the packetQueue. The maxFreeSlots
// value should reflect the max number of HTLC's that we're allowed to have
// outstanding within the commitment transaction.
func newPacketQueue(maxFreeSlots int, maxQueueLen int32) *packetQueue {
	p := &packetQueue{
		outgoingPkts: make(chan *htlcPacket),
		freeSlots:    make(chan struct{}, maxFreeSlots),
		quit:         make(chan struct{}),
		maxQueueLen:  maxQueueLen,
	}
	p.queueCond = sync.NewCond(&p.queueMtx)

	return p
}

// Start starts all goroutines that packetQueue needs to perform its normal
// duties.
func (p *packetQueue) Start() {
	p.wg.Add(1)
	go p.packetCoordinator()
}

// Stop signals the packetQueue for a graceful shutdown, and waits for all
// goroutines to exit.
func (p *packetQueue) Stop() {
	close(p.quit)

	// Now that we've closed the channel, we'll repeatedly signal the msg
	// consumer until we've detected that it has exited.
	for atomic.LoadInt32(&p.streamShutdown) == 0 {
		p.queueCond.Signal()
		time.Sleep(time.Millisecond * 100)
	}
}

// packetCoordinator is a goroutine that handles the packet overflow queue.
// Using a synchronized queue, outside callers are able to append to the end of
// the queue, waking up the coordinator when the queue transitions from empty
// to non-empty. The packetCoordinator will then aggressively try to empty out
// the queue, passing new htlcPackets to the channelLink as slots within the
// commitment transaction become available.
//
// Future iterations of the packetCoordinator will implement congestion
// avoidance logic in the face of persistent htlcPacket back-pressure.
//
// TODO(roasbeef): later will need to add back pressure handling heuristics
// like reg congestion avoidance:
//   * random dropping, RED, etc
func (p *packetQueue) packetCoordinator() {
	defer atomic.StoreInt32(&p.streamShutdown, 1)

	for {
		// First, we'll check our condition. If the queue of packets is
		// empty, then we'll wait until a new item is added.
		p.queueCond.L.Lock()
		for len(p.queue) == 0 {
			p.queueCond.Wait()

			// If we were woke up in order to exit, then we'll do
			// so. Otherwise, we'll check the message queue for any
			// new items.
			select {
			case <-p.quit:
				p.queueCond.L.Unlock()
				return
			default:
			}
		}
		p.queueCond.L.Unlock()

		// If there aren't any further messages to sent (or the link
		// didn't immediately read our message), then we'll block and
		// wait for a new message to be sent into the overflow queue,
		// or for the link's htlcForwarder to wake up.
		select {
		case <-p.freeSlots:
			debug_print("free slots indicated")
			if (SPIDER_FLAG) {
				// FIXME: just go to sleep for test.
				time.Sleep(time.Millisecond * 200)
			}
			// Pop item with highest priority from the front of the queue. This will
			// set us up for the next iteration. If the queue is empty at this point,
			// then we'll block at the top.
			// Note that the item must be retrived within queueCond lock as any new
			// inserted item might have a higher priority than p.queue[0]
			p.queueCond.L.Lock()
			nextPkt := p.queue[0].packet
			heap.Pop(&p.queue)
			p.queueCond.L.Unlock()

			select {
			case p.outgoingPkts <- nextPkt:
				// Only decrease the queueLen and totalHtlcAmt once the packet has been
				// sent out
				debug_print("going to remove nextPkt from the queue")
				atomic.AddInt32(&p.queueLen, -1)
				atomic.AddInt64(&p.totalHtlcAmt, int64(-nextPkt.amount))
			case <-p.quit:
				return
			}

		case <-p.quit:
			return

		default:
		}
	}
}

// AddPkt adds the referenced packet to the overflow queue, preserving ordering
// of the existing items.
func (p *packetQueue) AddPkt(pkt *htlcPacket) {
	// First, we'll lock the condition, and add the message to the end of
	// the message queue, and increment the internal atomic for tracking
	// the queue's length.
	p.queueCond.L.Lock()
	if atomic.LoadInt32(&p.queueLen) < p.maxQueueLen {
		heap.Push(&p.queue, makeNode(pkt))
		atomic.AddInt32(&p.queueLen, 1)
		atomic.AddInt64(&p.totalHtlcAmt, int64(pkt.amount))
	} else {
		log.Warnf("Packet %v dropped as overflow queue is full", pkt.incomingHTLCID)
	}
	p.queueCond.L.Unlock()

	// With the message added, we signal to the msgConsumer that there are
	// additional messages to consume.
	p.queueCond.Signal()
}

// SignalFreeSlot signals to the queue that a new slot has opened up within the
// commitment transaction. The max amount of free slots has been defined when
// initially creating the packetQueue itself. This method, combined with AddPkt
// creates the following abstraction: a synchronized queue of infinite length
// which can be added to at will, which flows onto a commitment of fixed
// capacity.
func (p *packetQueue) SignalFreeSlot() {
	// We'll only send over a free slot signal if the queue *is not* empty.
	// Otherwise, it's possible that we attempt to overfill the free slots
	// semaphore and block indefinitely below.
	debug_print("signalFree slot!")
	debug_print(fmt.Sprintf("queue len is %d\n", p.queueLen))
	if atomic.LoadInt32(&p.queueLen) == 0 {
		return
	}

	select {
	case p.freeSlots <- struct{}{}:
	case <-p.quit:
		debug_print("they made us quit instead of signal free slot")
		return
	}
}

// Length returns the number of pending htlc packets present within the over
// flow queue.
func (p *packetQueue) Length() int32 {
	return atomic.LoadInt32(&p.queueLen)
}

// TotalHtlcAmount is the total amount (in mSAT) of all HTLC's currently
// residing within the overflow queue.
func (p *packetQueue) TotalHtlcAmount() lnwire.MilliSatoshi {
	// TODO(roasbeef): also factor in fee rate?
	return lnwire.MilliSatoshi(atomic.LoadInt64(&p.totalHtlcAmt))
}
