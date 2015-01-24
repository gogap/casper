package main

import (
	"flag"

	"github.com/gogap/casper"
)

func main() {
	flag.Parse()

	casper.BuildApp("./casper.conf.example")

	casper.GetAppByName("example").Run()
}
