package main

import (
	"fmt"

	"github.com/gogap/casper"
)

func main() {
	msg, _ := casper.NewComponentMessage("", nil)
	msg.Payload.SetContext(casper.REQ_X_API, "demo")

	reply, err := casper.CallService("zmq", "tcp://127.0.0.1:5555", msg)
	if err != nil {
		fmt.Println(err)
		return
	}

	replys, _ := reply.Serialize()
	fmt.Println(string(replys))
}
