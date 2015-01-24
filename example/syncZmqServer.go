package main

import (
	"flag"

	"github.com/gogap/casper"
)

func main() {
	flag.Parse()

	casper.BuildApp("./casper.conf.example")

	com := casper.GetAppByName("syncService")
	com.SetHandler(handler)
	com.Run()
}

func handler(msg *casper.Payload) (result interface{}, err error) {

	rst := &struct {
		Name string
		Age int
	}{
		Name: "小明",
		Age: 6}
	
	return rst, nil
}
