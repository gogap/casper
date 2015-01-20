package utils

import (
	"github.com/gogap/casper"
	"time"
)

func IamWorking() {
	for {
		time.Sleep(1 * time.Second)
	}
}

func CallService(serviceType, addr string, msg *casper.ComponentMessage) {
	switch serviceType {
	case "zmq":
		{
			casper.zmq
		}
	}
}
