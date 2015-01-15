package main

import (
	"flag"
	
	"github.com/gogap/casper"
	"github.com/gogap/casper/utils"
)

func main() {
	flag.Parse()
	
	casper.BuildComFromConfig("component.conf.example")
	casper.GetComponentByName("com1").Run()
	utils.IamWorking()
}


