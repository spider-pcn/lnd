package htlcswitch

import (
	"fmt"
)

// FIXME: temporary until we can use the global flags
var SPIDER_FLAG bool = true
var DEBUG_FLAG bool = false
var LOG_FIREBASE bool = false
// multiply MAX_HTLC count by this to get max overflowQueue length
var SPIDER_QUEUE_LENGTH_SCALE int = 8

func debug_print(str string) {
	if (DEBUG_FLAG) {
		fmt.Printf(str)
	}
}
