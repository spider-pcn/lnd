package htlcswitch

import (
	"fmt"
	"os"
	"hash/fnv"
)

// FIXME: temporary until we can use the global flags
var SPIDER_FLAG bool = os.Getenv("SPIDER_QUEUE") == "1"
var LP_ROUTING bool = os.Getenv("SPIDER_LP_ROUTING") == "1"

var DEBUG_FLAG bool = false
var LOG_FIREBASE bool = os.Getenv("SPIDER_LOG_FIREBASE") == "1"
var FIREBASE_URL string = "https://spider2.firebaseio.com/"

// globals required for LP routing 
var ETA float32 = 0.5
var KAPPA float32 = 0.5
var T_UPDATE float32 = 1.00		// in terms of seconds 
var DELTA float32 = 1.00 // RTT of longest path, in seconds

// multiply MAX_HTLC count by this to get max overflowQueue length
var SPIDER_QUEUE_LENGTH_SCALE int = 8
var FILENAME string = "./log_test.txt"
var EXP_NAME string = os.Getenv("SPIDER_EXP_NAME")
// time in ms
var UPDATE_INTERVAL int = 1000

func hash(s string) uint32 {
        h := fnv.New32a()
        h.Write([]byte(s))
        return h.Sum32()
}

func debug_print(str string) {
	if (DEBUG_FLAG) {
		//fmt.Printf(str)
		f, err := os.OpenFile(FILENAME, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}

		defer f.Close()

		if _, err = f.WriteString(str); err != nil {
			panic(err)
		}
	}
}

func set_exp_name(name string) {
	fmt.Println("set exp name!: " + name)
	EXP_NAME = name
}
