package htlcswitch

import (
	"fmt"
	"os"
	"hash/fnv"
)

// FIXME: temporary until we can use the global flags
var SPIDER_FLAG bool = true
var DEBUG_FLAG bool = true
var LOG_FIREBASE bool = true
// multiply MAX_HTLC count by this to get max overflowQueue length
var SPIDER_QUEUE_LENGTH_SCALE int = 8
var FILENAME string = "./log_test.txt"
var EXP_NAME string = "DEBUG"
var SWITCH_NAME string = "DEFAULT"
var UPDATE_INTERVAL int = 10

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

func get_switch_name() string {
	return SWITCH_NAME;
}

func set_switch_name(name string) {
	fmt.Println("set switch name!: " + name)
	SWITCH_NAME = name;
}
