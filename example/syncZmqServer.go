package main

import (
	"flag"

	"github.com/gogap/casper"
)

func main() {
	flag.Parse()

	casper.BuildAppFromConfigFile("./casper.conf.example")

	com := casper.GetAppByName("syncService")
	com.SetHandler(handler)
	com.Run()
}

func handler(msg *casper.Payload) (result interface{}, err error) {
	str := "this is syncService self"
	return &str, nil
}
