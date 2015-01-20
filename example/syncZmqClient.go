package main

import (
	"fmt"
	
	"github.com/gogap/casper"
)

func main() {
	msg, _ := casper.NewComponentMessage("")
	msg.Payload.SetContext(casper.REQ_X_API, "demo")

	reply, _ :=casper.CallService("zmq", "tcp://localhost:5555", msg)

	replys, _ := reply.Serialize()
	fmt.Println(string(replys))
}
