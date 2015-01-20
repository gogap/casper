package main

import (
	"fmt"
	
	"github.com/gogap/casper"
)

func main() {
	msg, _ := casper.NewComponentMessage("")
	msg.Payload.SetContext(casper.REQ_X_API, "demo")

	req, _ := msg.Serialize()

	reply, _ := casper.ZmqSyncCall("tcp://localhost:5555", req)

	fmt.Println(string(reply))
}
