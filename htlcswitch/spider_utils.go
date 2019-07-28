package htlcswitch

import (
	"fmt"
	"hash/fnv"
	"os"
	"strconv"
)

// FIXME: temporary until we can use the global flags
var SPIDER_FLAG bool = os.Getenv("SPIDER_QUEUE") == "1"
var LP_ROUTING bool = os.Getenv("SPIDER_LP_ROUTING") == "1"
var DCTCP bool = os.Getenv("SPIDER_DCTCP_ROUTING") == "1"
var TIMEOUT bool = os.Getenv("SPIDER_TIMEOUT") == "1"

var DEBUG_FLAG bool = false
var LOG_FIREBASE bool = os.Getenv("SPIDER_LOG_FIREBASE") == "1"

//var FIREBASE_URL string = "https://spider2.firebaseio.com/"
var FIREBASE_URL string = "https://spider3-b4420.firebaseio.com/"

// globals required for LP routing
var ETA, err = strconv.ParseFloat(os.Getenv("ETA"), 64)          //0.5
var KAPPA, errKappa = strconv.ParseFloat(os.Getenv("Kappa"), 64) //0.5
var XI, errXi = strconv.ParseFloat(os.Getenv("XI"), 64)          //1

// globals for DCTCP, ALPHA BETA defined in rouyting/router.go
var QUEUE_THRESHOLD, errQT = strconv.ParseFloat(os.Getenv("QUEUE_THRESHOLD"), 64) // 5.00

// same as measurement_interval
var T_UPDATE, errT = strconv.ParseFloat(os.Getenv("XI"), 64)                              //  1.5 in terms of seconds
var QUEUE_DRAIN_TIME, errQ = strconv.ParseFloat(os.Getenv("QUEUE_DRAIN_TIME"), 64)        // 5.00
var SERVICE_ARRIVAL_WINDOW, errWindow = strconv.Atoi(os.Getenv("SERVICE_ARRIVAL_WINDOW")) // 300

// multiply MAX_HTLC count by this to get max overflowQueue length
var SPIDER_QUEUE_LENGTH_SCALE int = 8
var FILENAME string = "./log_test.txt"
var EXP_NAME string = os.Getenv("SPIDER_EXP_NAME")

// time in ms
var STATS_INTERVAL, errStats = strconv.Atoi(os.Getenv("STATS_INTERVAL")) // 1000

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func debug_print(str string) {
	if DEBUG_FLAG {
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
