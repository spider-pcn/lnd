package htlcswitch

import (
		"testing"
		"time"
		"github.com/btcsuite/btcutil"
		"github.com/lightningnetwork/lnd/lnwire"
		"fmt"
		"github.com/lightningnetwork/lnd/lnpeer"
)

/// Helper function that manages boilerplate code for sending payment from
/// @sendingPeer ->  @receivingPeer through the given @path. In turn, it calls
/// helper functions defined in htlcswitch/test_utils.go, but these functions
/// aren't structured very intuitively, at least for the cases spider is
/// testing so far. So this is a slightly nicer wrapper.
/// Other args:
///   @n: testing network created using helper utility newThreeHopNetwork
///		@delaySeconds: sleep at the start so we can test various scenarios.
///		@err_ch: returns the error value on this channel
func SendMoneyWithDelay(n *threeHopNetwork, satoshis btcutil.Amount,
			sendingPeer, receivingPeer lnpeer.Peer, delaySeconds int64, err_ch chan
			error, path ...*channelLink) {
	time.Sleep(time.Duration(delaySeconds) * time.Second)
	amount := lnwire.NewMSatFromSatoshis(satoshis)
	htlcAmt, totalTimelock, hops := generateHops(amount, testStartingHeight,
			path...)
	firstHop := path[0].ShortChanID()
	// waits 10 seconds for timeout. this seems more than enough for any of the
	// test cases we have currently.
	_, err := n.makePayment(
		sendingPeer, receivingPeer, firstHop, hops, amount, htlcAmt,
		totalTimelock,
	).Wait(10 * time.Second)
	// send err back over the channel.
	err_ch <- err
}
/// Wrapper function that just calls helper utilities from
/// htlcswitch/test_utils.go to setup the network approrpriately.
func StartThreeHopNetwork(firstChanBal, secondChanBal int64, t *testing.T) (*threeHopNetwork, func()) {
	// creates Alice -> Bob -> Carol bidirectional channels with the given
	// satoshis for each direction of the channel,
	channels, cleanUp, _, err := createClusterChannels(
		btcutil.Amount(btcutil.SatoshiPerBitcoin*firstChanBal),
		btcutil.Amount(btcutil.SatoshiPerBitcoin*secondChanBal))
	if err != nil {
		t.Fatalf("unable to create channel: %v", err)
	}
	n := newThreeHopNetwork(t, channels.aliceToBob, channels.bobToAlice,
	channels.bobToCarol, channels.carolToBob, testStartingHeight)
	if err := n.start(); err != nil {
		t.Fatalf("unable to start three hop network: %v", err)
	}
	return n, cleanUp
}

// bob<->alice channel has insufficient BTC capacity/bandwidth. In this test we
// send the payment from Alice to Carol over the Bob peer. (Alice -> Bob ->
// Carol) with the Bob -> Carol link being the bottleneck. This is put in a
// queue in the Bob -> Carol channel. After Carol sends Bob money too, this
// payment should go through.
func TestSpiderTemporarilyInsufficientFunds(t *testing.T) {
	t.Parallel()
	n, cleanUp := StartThreeHopNetwork(5, 3, t)
	defer cleanUp()
	defer n.stop()
	// First, we will send money from Alice -> Carol. This should block on Bob ->
	// Carol due to insufficient funds.
	// What we expect to happen:
	// * HTLC add request to be sent to from Alice to Bob.
	// * Alice<->Bob commitment states to be updated.
	// * Bob trying to add HTLC add request in Bob<->Carol channel.
	// * Not enough funds for that, so it gets added to overFlowQueue in
	// Bob->Carol channel.
	// * Carol sends Bob money.
	// * Bob -> Carol channel now has enough money, and the Bob -> Carol
	// payment should succeed, thereby letting the Alice -> Carol payment to
	// succeed as well.
	c := make(chan error)
	go SendMoneyWithDelay(n, 4 * btcutil.SatoshiPerBitcoin, n.aliceServer,
			n.carolServer, 0, c, n.firstBobChannelLink, n.carolChannelLink)
	// send money on carol -> bob after a 2 seconds delay
	go SendMoneyWithDelay(n, 2 * btcutil.SatoshiPerBitcoin, n.carolServer,
			n.bobServer, 2, make(chan error), n.secondBobChannelLink)

	// we only need to check if alice -> carol payment succeeded or not. Since
	// initially there wasn't enough money on the bob -> carol channel, this
	// could only have succeeded if the carol -> bob payment happened in time and
	// the overflowQueue was working properly.
	firstPaymentErr := <-c
	if firstPaymentErr != nil {
		fmt.Println(firstPaymentErr)
		t.Fatal("error has been received in first payment, alice->bob")
	}
}

// bob<->alice channel has insufficient BTC capacity/bandwidth. In this test we
// send the payment from Alice to Carol over the Bob peer. (Alice -> Bob ->
// Carol) with the Bob -> Carol link being the bottleneck.  Here, Carol does
// not send Bob enough money, and this payment should eventually timeout.
func TestSpiderInsufficientFunds(t *testing.T) {
	t.Parallel()
	n, cleanUp := StartThreeHopNetwork(5, 3, t)
	defer cleanUp()
	defer n.stop()
	// First, we will send money from Alice -> Carol. This should block on Bob ->
	// Carol due to insufficient funds.
	// What we expect to happen:
	// * HTLC add request to be sent to from Alice to Bob.
	// * Alice<->Bob commitment states to be updated.
	// * Bob trying to add HTLC add request in Bob<->Carol channel.
	// * Not enough funds for that, so it gets added to overFlowQueue in
	// Bob->Carol channel.
	// * Carol sends Bob money, but it is not enough to cover Alice -> Carol
	// payment.
	// * Alice -> Carol payment should time out.
	c := make(chan error)
	go SendMoneyWithDelay(n, 4 * btcutil.SatoshiPerBitcoin, n.aliceServer, n.carolServer, 0, c, n.firstBobChannelLink, n.carolChannelLink)
	// carol -> bob after a 2 seconds delay
	go SendMoneyWithDelay(n, 0.9 * btcutil.SatoshiPerBitcoin, n.carolServer, n.bobServer, 2, make(chan error), n.secondBobChannelLink)

	firstPaymentErr := <-c
	// Since not enough money was transfered from Carol -> Bob, the Alice ->
	// Carol payment did not have enough funds even at the end and should time
	// out. Thus an error must be returned, so we check to ensure error isn't
	// nil.
	if firstPaymentErr == nil {
		fmt.Println(firstPaymentErr)
		t.Fatal("no error was received in the payment, alice->carol, despite not having enough funds")
	}
}

// bob<->alice channel has insufficient BTC capacity/bandwidth. In this test we
// send the payment from Alice to Carol over the Bob peer. (Alice -> Bob ->
// Carol) with the Bob -> Carol link being the bottleneck.
// Here, Carol sends Bob money in partial segments, and eventually when enough
// has been sent, the Alice -> Carol link should succeed.
func TestSpiderTemporarilyInsufficientFundsMultiplePayments (t *testing.T) {
	t.Parallel()
	n, cleanUp := StartThreeHopNetwork(5, 3, t)
	defer cleanUp()
	defer n.stop()

	// What we expect to happen:
	// * HTLC add request to be sent to from Alice to Bob.
	// * Alice<->Bob commitment states to be updated.
	// * Bob trying to add HTLC add request in Bob<->Carol channel.
	// * Not enough funds for that, so it gets added to overFlowQueue in
	// Bob->Carol channel.
	// * Carol sends Bob money. Still not enough funds to forward Alice's
	// request.
	// * Carol keeps sending Bob more money.
	// * Eventually, Bob -> Carol channel now has enough money, and the Bob -> Carol
	// payment should succeed, thereby letting the Alice -> Carol payment to
	// succeed as well.
	c := make(chan error)
	go SendMoneyWithDelay(n, 4 * btcutil.SatoshiPerBitcoin, n.aliceServer, n.carolServer, 0, c, n.firstBobChannelLink, n.carolChannelLink)
	// carol -> bob after a 2 seconds delay
	for i := 0; i < 4; i++ {
		go SendMoneyWithDelay(n, 0.3 * btcutil.SatoshiPerBitcoin, n.carolServer, n.bobServer, int64(i+1), make(chan error), n.secondBobChannelLink)
	}

	// we only need to check if alice -> carol payment succeeded or not. Since
	// initially there wasn't enough money on the bob -> carol channel, this
	// could only have succeeded if all the carol -> bob payments happened in
	// time and the overflowQueue was working properly.
	firstPaymentErr := <-c
	if firstPaymentErr != nil {
		fmt.Println(firstPaymentErr)
		t.Fatal("error has been received in first payment, alice->bob")
	}
}

// Long running flow just to test visualization.
// Not running this as part of the usual testing sequence, because the main
// test occurs with the visualization etc. which is managed separately with
// python scripts. For now, the creation of the MockServer relies on the
// testing.T argument "t", so adding it as a test here, but can work on
// removing that dependency from newThreeHopNetwork(...)
// To run this: change the name of the test to start with "Test", and then just
// run the test as usual with go test -run "RegexOfTestToRun"
func TestSpiderLongRunningFlow (t *testing.T) {
	NUM_PAYMENTS := 10000
	t.Parallel()
	n, cleanUp := StartThreeHopNetwork(50000, 50000, t)
	defer cleanUp()
	defer n.stop()
	c := make(chan error)
	for i := 0; i < NUM_PAYMENTS; i++ {
		go SendMoneyWithDelay(n, 1 * btcutil.SatoshiPerBitcoin, n.aliceServer, n.carolServer, 0, c, n.firstBobChannelLink, n.carolChannelLink)
		time.Sleep(1 * time.Second)
	}
	// wait for all the errors from the payments we initiated (all should be nil)
	for i := 0; i < NUM_PAYMENTS; i++ {
		<-c
	}
}

/// Trying to see how many payments can be sent from Alice -> Bob on the
/// given machine (basically, if the CPU can be a bottleneck on the
/// throughput).
func SpiderThroughput (t *testing.T) {
	NUM_PAYMENTS := 10
	t.Parallel()
	n, cleanUp := StartThreeHopNetwork(50000, 50000, t)
	defer cleanUp()
	defer n.stop()
	c := make(chan error)

	var startTime = time.Now()
	for i := 0; i < NUM_PAYMENTS; i++ {
		go SendMoneyWithDelay(n, 0.01 * btcutil.SatoshiPerBitcoin, n.aliceServer, n.bobServer, 0, c, n.firstBobChannelLink)
	}
	duration :=  time.Since(startTime)
	fmt.Printf("generating all the payments took: %s\n", duration)
	ms := float64(duration / time.Millisecond)
	fmt.Printf("throughput: %f/s\n", (float64(NUM_PAYMENTS) / (float64(ms)/1000.00)))

	for i := 0; i < NUM_PAYMENTS; i++ {
		<-c
	}
	duration =  time.Since(startTime)
	fmt.Printf("completing all payments took: %s\n", duration)
}
