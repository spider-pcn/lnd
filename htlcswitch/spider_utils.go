package htlcswitch

import (
	//"fmt"
	"os"
)

// FIXME: temporary until we can use the global flags
var SPIDER_FLAG bool = false
var DEBUG_FLAG bool = false
var LOG_FIREBASE bool = false
// multiply MAX_HTLC count by this to get max overflowQueue length
var SPIDER_QUEUE_LENGTH_SCALE int = 8
var FILENAME string = "./log_test.txt"

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
