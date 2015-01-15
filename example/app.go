package main


import (
	"flag"

	"github.com/gogap/casper"
)

func main() {
	flag.Parse()
	
	casper.BuildAppFromConfig("./casper.conf.example")

	casper.GetAppByName("example").Run()
}
