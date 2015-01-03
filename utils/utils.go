package utils

import (
	"encoding/json"
	"net/http"

	"github.com/cascades-fbp/cascades/runtime"
	"github.com/gogap/base_component"
	zmq "github.com/pebbe/zmq4"
)

func RequestToComponentMessage(request *http.Request) (msg *base_component.ComponentMessage) {
	return
}

// Create a ZMQ PULL socket & bind to a given endpoint
func CreateInputPort(endpoint string) (socket *zmq.Socket, err error) {
	socket, err = zmq.NewSocket(zmq.PULL)
	if err != nil {
		return nil, err
	}
	err = socket.Bind(endpoint)
	if err != nil {
		return nil, err
	}
	return socket, nil
}

// Create a ZMQ PUSH socket & connect to a given endpoint
func CreateOutputPort(endpoint string) (socket *zmq.Socket, err error) {
	socket, err = zmq.NewSocket(zmq.PUSH)
	if err != nil {
		return nil, err
	}
	err = socket.Connect(endpoint)
	if err != nil {
		return nil, err
	}
	return socket, nil
}

// Converts a given request to IP
func ComponentMessage2IP(msg *base_component.ComponentMessage) (ip [][]byte, err error) {
	if payload, e := json.Marshal(msg); e != nil {
		err = e
		return
	} else {
		ip = runtime.NewPacket(payload)
	}
	return
}

// Converts a given IP to response structure
func IP2ComponentMessage(ip [][]byte) (msg *base_component.ComponentMessage, err error) {
	err = json.Unmarshal(ip[1], &msg)
	return
}
