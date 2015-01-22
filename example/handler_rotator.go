package main

import (
	"flag"
	"fmt"

	"github.com/gogap/casper"
	"github.com/gogap/casper/utils"
)

type Mock struct {
}

type Product struct {
}

func (p *Mock) Handler(payload *casper.Payload) (interface{}, error) {
	fmt.Println("hello, i am mock")
	return nil, nil
}

func (p *Product) Handler(payload *casper.Payload) (interface{}, error) {
	fmt.Println("hello, i am prduct")
	return nil, nil
}

func main() {
	flag.Parse()

	casper.BuildHandlerRotatorConfig("handler_rotator.conf.example")
	casper.BuildComFromConfig("component.conf.example")

	casper.GetComponentByName("com1").Run()

	mock := Mock{}
	product := Product{}
	com4Handler := casper.NewHandlerRotator("com4", casper.RotatorParams{"mock": mock.Handler, "product": product.Handler})

	com4 := casper.GetComponentByName("com4")
	com4.SetHandler(com4Handler.Handler)
	com4.Run()
	utils.IamWorking()
}
