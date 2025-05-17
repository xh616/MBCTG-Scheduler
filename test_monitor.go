package main

import (
	"MBCTG/pkg/utils"
)

func main() {
	_ = utils.PrintNodeMonitorToRead("cpu")
	_ = utils.PrintNodeMonitorToRead("mem")
	_ = utils.PrintPodMonitorToRead("cpu", "demo1")
	_ = utils.PrintPodMonitorToRead("mem", "demo1")
}
