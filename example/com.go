package main

import (
	"flag"
	"fmt"

	"github.com/gogap/casper"
	"github.com/gogap/casper/utils"
)

func main() {
	flag.Parse()

	casper.BuildComponent("component.conf.example")
	casper.GetComponentByName("com1").SetHandler(handler).Run()
	utils.IamWorking()
}

func handler(msg *casper.Payload) (result interface{}, err error) {
	fmt.Println(">>>", msg)

	rst := &struct {
		Name string
		Age  int
	}{
		Name: "小明",
		Age:  6}

	return rst, nil
}
