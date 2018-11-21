package htlcswitch

import (
		"testing"
		"time"
		"github.com/btcsuite/btcutil"
		"github.com/lightningnetwork/lnd/lnwire"
		"fmt"
)

// bob<->alice channel has insufficient BTC capacity/bandwidth. In this test we
// send the payment from Alice to Carol over the Bob peer. (Alice -> Bob ->
// Carol) with the Bob -> Carol link being the bottleneck.
// After Carol sends Bob money too, this payment should go through.
func TestSpiderTemporarilyInsufficentFunds(t *testing.T) {
	t.Parallel()

	channels, cleanUp, _, err := createClusterChannels(
		btcutil.SatoshiPerBitcoin*5,
		btcutil.SatoshiPerBitcoin*3)
		if err != nil {
			t.Fatalf("unable to create channel: %v", err)
		}
		defer cleanUp()

		n := newThreeHopNetwork(t, channels.aliceToBob, channels.bobToAlice,
		channels.bobToCarol, channels.carolToBob, testStartingHeight)
		if err := n.start(); err != nil {
			t.Fatalf("unable to start three hop network: %v", err)
		}

		defer n.stop()

		go func() {
			fmt.Println("in the carol->bob payment routine. Will sleep first")
			time.Sleep(1 * time.Second)
			fmt.Println("woke up in the carol->bob payment routine");
			amount := lnwire.NewMSatFromSatoshis(2 * btcutil.SatoshiPerBitcoin)
			htlcAmt, totalTimelock, hops := generateHops(amount, testStartingHeight,
					n.secondBobChannelLink)

			firstHop := n.secondBobChannelLink.ShortChanID()
			_, err := n.makePayment(
				n.carolServer, n.bobServer, firstHop, hops, amount, htlcAmt,
				totalTimelock,
			).Wait(30 * time.Second)

			if err != nil {
				t.Fatal("carol->bob failed")
			}
			fmt.Println("carol->bob successful")
		}()

		amount := lnwire.NewMSatFromSatoshis(4 * btcutil.SatoshiPerBitcoin)
		htlcAmt, totalTimelock, hops := generateHops(amount, testStartingHeight,
					n.firstBobChannelLink, n.carolChannelLink)

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

		firstHop := n.firstBobChannelLink.ShortChanID()
		// launch second payment here, which sleeps for a bit, and then pays from
		// carol to bob.
		_, err = n.makePayment(
			n.aliceServer, n.carolServer, firstHop, hops, amount, htlcAmt,
			totalTimelock,
		).Wait(80 * time.Second)
		// modifications on existing test. We do not want it to fail here.
		if err != nil {
			fmt.Println(err)
			t.Fatal("error has been received in first payment, alice->bob")
		}
		fmt.Println("alice -> carol successful")
		// if we reach this point, then all the payments have succeeded.
}

// bob<->alice channel has insufficient BTC capacity/bandwidth. In this test we
// send the payment from Alice to Carol over the Bob peer. (Alice -> Bob ->
// Carol) with the Bob -> Carol link being the bottleneck.
// Here, Carol does not send Bob any money, and this payment should eventually
// timeout.
func TestSpiderInsufficentFunds(t *testing.T) {
	t.Parallel()

	channels, cleanUp, _, err := createClusterChannels(
		btcutil.SatoshiPerBitcoin*5,
		btcutil.SatoshiPerBitcoin*3)
		if err != nil {
			t.Fatalf("unable to create channel: %v", err)
		}
		defer cleanUp()

		n := newThreeHopNetwork(t, channels.aliceToBob, channels.bobToAlice,
		channels.bobToCarol, channels.carolToBob, testStartingHeight)
		if err := n.start(); err != nil {
			t.Fatalf("unable to start three hop network: %v", err)
		}
		defer n.stop()

		amount := lnwire.NewMSatFromSatoshis(4 * btcutil.SatoshiPerBitcoin)
		htlcAmt, totalTimelock, hops := generateHops(amount, testStartingHeight,
					n.firstBobChannelLink, n.carolChannelLink)

		// What we expect to happen:
		// * HTLC add request to be sent to from Alice to Bob.
		// * Alice<->Bob commitment states to be updated.
		// * Bob trying to add HTLC add request in Bob<->Carol channel.
		// * Not enough funds for that, so it gets added to overFlowQueue in
		// Bob->Carol channel.
		// Payment times out.

		firstHop := n.firstBobChannelLink.ShortChanID()
		// launch second payment here, which sleeps for a bit, and then pays from
		// carol to bob.
		_, err = n.makePayment(
			n.aliceServer, n.carolServer, firstHop, hops, amount, htlcAmt,
			totalTimelock,
		).Wait(10 * time.Second)
		// modifications on existing test. We do not want it to fail here.
		if err == nil {
			fmt.Println(err)
			t.Fatal("no error was received in the payment, alice->bob")
		}
		fmt.Println("alice -> carol failed succesfully")
		// if we reach this point, then all the payments have succeeded.
}

// bob<->alice channel has insufficient BTC capacity/bandwidth. In this test we
// send the payment from Alice to Carol over the Bob peer. (Alice -> Bob ->
// Carol) with the Bob -> Carol link being the bottleneck.
// Here, Carol sends Bob money in partial segments, and eventually when enough
// has been sent, the Alice -> Carol link should succeed.
func TestSpiderTemporarilyInsufficentFundsMultiplePayments (t *testing.T) {
	t.Parallel()

	channels, cleanUp, _, err := createClusterChannels(
		btcutil.SatoshiPerBitcoin*5,
		btcutil.SatoshiPerBitcoin*3)
		if err != nil {
			t.Fatalf("unable to create channel: %v", err)
		}
		defer cleanUp()

		n := newThreeHopNetwork(t, channels.aliceToBob, channels.bobToAlice,
		channels.bobToCarol, channels.carolToBob, testStartingHeight)
		if err := n.start(); err != nil {
			t.Fatalf("unable to start three hop network: %v", err)
		}

		defer n.stop()

		go func() {
			fmt.Println("in the carol->bob payment routine. Will sleep first")
			time.Sleep(1 * time.Second)
			fmt.Println("woke up in the carol->bob payment routine")
			for i := 0; i < 4; i++ {
				amount := lnwire.NewMSatFromSatoshis(0.5 * btcutil.SatoshiPerBitcoin)
				htlcAmt, totalTimelock, hops := generateHops(amount, testStartingHeight,
						n.secondBobChannelLink)

				firstHop := n.secondBobChannelLink.ShortChanID()
				_, err := n.makePayment(
					n.carolServer, n.bobServer, firstHop, hops, amount, htlcAmt,
					totalTimelock,
				).Wait(30 * time.Second)

				if err != nil {
					t.Fatal("carol->bob failed")
				}
				fmt.Println("carol->bob successful try ")
				time.Sleep(1*time.Second)
			}
		}()

		amount := lnwire.NewMSatFromSatoshis(4 * btcutil.SatoshiPerBitcoin)
		htlcAmt, totalTimelock, hops := generateHops(amount, testStartingHeight,
					n.firstBobChannelLink, n.carolChannelLink)

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

		firstHop := n.firstBobChannelLink.ShortChanID()
		// launch second payment here, which sleeps for a bit, and then pays from
		// carol to bob.
		_, err = n.makePayment(
			n.aliceServer, n.carolServer, firstHop, hops, amount, htlcAmt,
			totalTimelock,
		).Wait(80 * time.Second)
		// modifications on existing test. We do not want it to fail here.
		if err != nil {
			fmt.Println(err)
			t.Fatal("error has been received in first payment, alice->bob")
		}
		fmt.Println("alice -> carol successful")
		// if we reach this point, then all the payments have succeeded.
		// Sleep to let all intermediate payments Carol -> bob succeed.
		time.Sleep(5*time.Second)
}

// Long running flow just to test visualization.
func LongRunningFlow (t *testing.T) {
	t.Parallel()

	channels, cleanUp, _, err := createClusterChannels(
		btcutil.SatoshiPerBitcoin*500000,
		btcutil.SatoshiPerBitcoin*300000)
		if err != nil {
			t.Fatalf("unable to create channel: %v", err)
		}
		defer cleanUp()

		n := newThreeHopNetwork(t, channels.aliceToBob, channels.bobToAlice,
		channels.bobToCarol, channels.carolToBob, testStartingHeight)
		if err := n.start(); err != nil {
			t.Fatalf("unable to start three hop network: %v", err)
		}

		defer n.stop()
		// temporary:
		go func() {
			fmt.Println("in the bob->carol payment routine. Will sleep first")
			time.Sleep(2 * time.Second)
			fmt.Println("woke up in the bob->carol payment routine");
			for i := 0; i < 10000; i++ {
				fmt.Println(i)
				amount := lnwire.NewMSatFromSatoshis(1 * btcutil.SatoshiPerBitcoin)
				htlcAmt, totalTimelock, hops := generateHops(amount, testStartingHeight,
						n.secondBobChannelLink)

				firstHop := n.carolChannelLink.ShortChanID()
				_, err := n.makePayment(
					n.bobServer, n.carolServer, firstHop, hops, amount, htlcAmt,
					totalTimelock,
				).Wait(30 * time.Second)

				if err != nil {
					t.Fatal("bob->carol failed")
				}
				fmt.Println("bob->carol successful")

				htlcAmt, totalTimelock, hops = generateHops(amount, testStartingHeight,
						n.firstBobChannelLink)

				firstHop = n.aliceChannelLink.ShortChanID()
				_, err = n.makePayment(
					n.bobServer, n.aliceServer, firstHop, hops, amount, htlcAmt,
					totalTimelock,
				).Wait(30 * time.Second)

				if err != nil {
					t.Fatal("bob->alice failed")
				}
				fmt.Println("bob->alice successful")
				time.Sleep(1 * time.Second)
			}
		}()

		go func() {
			fmt.Println("in the carol->bob payment routine. Will sleep first")
			time.Sleep(1 * time.Second)
			fmt.Println("woke up in the carol->bob payment routine");
			amount := lnwire.NewMSatFromSatoshis(2 * btcutil.SatoshiPerBitcoin)
			htlcAmt, totalTimelock, hops := generateHops(amount, testStartingHeight,
					n.secondBobChannelLink)

			firstHop := n.secondBobChannelLink.ShortChanID()
			_, err := n.makePayment(
				n.carolServer, n.bobServer, firstHop, hops, amount, htlcAmt,
				totalTimelock,
			).Wait(30 * time.Second)

			if err != nil {
				t.Fatal("carol->bob failed")
			}
			fmt.Println("carol->bob successful")
		}()

		amount := lnwire.NewMSatFromSatoshis(4 * btcutil.SatoshiPerBitcoin)
		htlcAmt, totalTimelock, hops := generateHops(amount, testStartingHeight,
					n.firstBobChannelLink, n.carolChannelLink)

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

		firstHop := n.firstBobChannelLink.ShortChanID()
		// launch second payment here, which sleeps for a bit, and then pays from
		// carol to bob.
		_, err = n.makePayment(
			n.aliceServer, n.carolServer, firstHop, hops, amount, htlcAmt,
			totalTimelock,
		).Wait(80 * time.Second)
		// modifications on existing test. We do not want it to fail here.
		if err != nil {
			fmt.Println(err)
			t.Fatal("error has been received in first payment, alice->bob")
		}
		// FIXME: temporary
		time.Sleep(1000 * time.Second)
		// if we reach this point, then all the payments have succeeded.
}


func NotTestSpiderThroughput (t *testing.T) {
	t.Parallel()
	var NUM_PAYMENTS = int(10000)
  // FIXME: do we even care about the second channel?
	channels, cleanUp, _, err := createClusterChannels(
		btcutil.SatoshiPerBitcoin*5000000,
		btcutil.SatoshiPerBitcoin*5000000)
		if err != nil {
			t.Fatalf("unable to create channel: %v", err)
		}
		defer cleanUp()
		n := newThreeHopNetwork(t, channels.aliceToBob, channels.bobToAlice,
				channels.bobToCarol, channels.carolToBob, testStartingHeight)
		if err := n.start(); err != nil {
			t.Fatalf("unable to start three hop network: %v", err)
		}
		defer n.stop()
		var startTime = time.Now()
		for i := 0; i < NUM_PAYMENTS; i++ {
			go func() {
				amount := lnwire.NewMSatFromSatoshis(0.01*btcutil.SatoshiPerBitcoin)
				htlcAmt, totalTimelock, hops := generateHops(amount, testStartingHeight,
									n.firstBobChannelLink)
				firstHop := n.firstBobChannelLink.ShortChanID()
				//totalTimelock = 108
				_, err := n.makePayment(
					n.aliceServer, n.bobServer, firstHop, hops, amount, htlcAmt,
					totalTimelock,
				).Wait(200 * time.Second)

				if err != nil {
					t.Fatal("alice->bob FAILED")
				}
			}()
		}
		duration :=  time.Since(startTime)
		fmt.Printf("generating all the payments took: %s\n", duration)
		ms := float64(duration / time.Millisecond)
		//fmt.Println(seconds)
		//fmt.Println(float64(seconds))
		fmt.Printf("throughput: %f/s\n", (float64(NUM_PAYMENTS) / (float64(ms)/1000.00)))
		time.Sleep(250 * time.Second)
}
