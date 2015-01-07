package main

import (
	. "github.com/gogap/casper"
)

func main() {
	BuildAppFromConfig("casper.conf.example")

	GetApp("example").Run()
}

