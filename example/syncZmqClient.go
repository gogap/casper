package main

import (
	"fmt"
	
	"github.com/gogap/casper"
)

func main() {
	msg, _ := casper.NewComponentMessage("")
	msg.Payload.SetContext(casper.REQ_X_API, "demo")

	reply, _ := casper.CallService("zmq", "tcp://localhost:5555", msg)

	fmt.Println(string(reply))
}
