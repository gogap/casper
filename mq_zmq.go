package casper

import (
	"fmt"

	zmq "github.com/pebbe/zmq4"
)

const componentPacket byte = 0x01

type mqZmq struct {
	url    string
	socket *zmq.Socket
}

func init() {
	registerMq("zmq", NewMqZmq)
}

func NewMqZmq(url string) mq {
	return &mqZmq{url: url, socket: nil}
}

func (p *mqZmq) Ready() (err error) {
	if p.url == "" {
		return fmt.Errorf("zmq's url nil")
	}
	p.socket, err = createZmqInputPort(p.url)
	return
}

func (p *mqZmq) RecvMessage() ([]byte, error) {
	ip, err := p.socket.RecvMessageBytes(0)
	if err != nil {
		return nil, err
	}

	if !(len(ip) == 2 && len(ip[0]) == 1 && ip[0][0] == componentPacket) {
		return ip[0], fmt.Errorf("recv not valid message")
	}

	return ip[1], nil
}

func (p *mqZmq) SendToNext(msg []byte) (total int, err error) {
	if p.socket == nil {
		p.socket, err = createZmqOutputPort(p.url)
		if err != nil {
			return 0, err
		}
	}

	packet := [][]byte{[]byte{componentPacket}, msg}
	return p.socket.SendMessage(packet)
}

// Create a ZMQ PULL socket & bind to a given endpoint
func createZmqInputPort(url string) (socket *zmq.Socket, err error) {
	if socket, err = zmq.NewSocket(zmq.PULL); err != nil {
		return nil, err
	}
	if err = socket.Bind(url); err != nil {
		return nil, err
	}

	return socket, nil
}

// Create a ZMQ PUSH socket & connect to a given endpoint
func createZmqOutputPort(url string) (socket *zmq.Socket, err error) {
	if socket, err = zmq.NewSocket(zmq.PUSH); err != nil {
		return nil, err
	}
	if err = socket.Connect(url); err != nil {
		return nil, err
	}

	return socket, nil
}
