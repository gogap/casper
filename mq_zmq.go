package casper

import (
	"github.com/gogap/errors"
	zmq "github.com/pebbe/zmq4"

	"github.com/gogap/casper/errorcode"
)

const componentPacket byte = 0x01

type mqZmq struct {
	url    string
	socket *zmq.Socket
}

func init() {
	registerMq("zmq", NewMqZmq)
}

func NewMqZmq(url string) MessageQueue {
	return &mqZmq{url: url, socket: nil}
}

func (p *mqZmq) Ready() (err error) {
	if p.url == "" {
		err = errorcode.ERR_ZMQ_URL_IS_EMPTY.New()
		return
	}
	p.socket, err = createZmqInputPort(p.url)
	return
}

func (p *mqZmq) RecvMessage() (msg []byte, err error) {
	var msgs [][]byte
	if msgs, err = p.socket.RecvMessageBytes(0); err != nil {
		err = errorcode.ERR_ZMQ_RECV_MSG_FAILED.New(
			errors.Params{
				"url": p.url,
				"err": err})

		return nil, err
	}

	if !isValidPacket(msgs) {
		err = errorcode.ERR_ZMQ_RECV_MSG_FAILED.New(
			errors.Params{"url": p.url})

		return nil, err
	}

	return msgs[1], nil
}

func (p *mqZmq) SendToNext(msg []byte) (total int, err error) {
	if p.socket == nil {
		p.socket, err = createZmqOutputPort(p.url)
		if err != nil {
			return 0, err
		}
	}

	packet := newPacket(msg)
	return p.socket.SendMessage(packet)
}

// Create a ZMQ PULL socket & bind to a given endpoint
func createZmqInputPort(url string) (socket *zmq.Socket, err error) {
	if socket, err = zmq.NewSocket(zmq.PULL); err != nil {
		err = errorcode.ERR_NEW_ZMQ_FAILED.New(
			errors.Params{
				"url":  url,
				"type": "PULL",
				"err":  err})

		return nil, err
	}

	if err = socket.Bind(url); err != nil {
		err = errorcode.ERR_ZMQ_COULD_NOT_BIND_URL.New(
			errors.Params{
				"url":  url,
				"type": "PULL",
				"err":  err})

		return nil, err
	}

	return socket, nil
}

// Create a ZMQ PUSH socket & connect to a given endpoint
func createZmqOutputPort(url string) (socket *zmq.Socket, err error) {
	if socket, err = zmq.NewSocket(zmq.PUSH); err != nil {
		err = errorcode.ERR_NEW_ZMQ_FAILED.New(
			errors.Params{
				"url":  url,
				"type": "PUSH",
				"err":  err})

		return nil, err
	}

	if err = socket.Connect(url); err != nil {
		err = errorcode.ERR_ZMQ_COULD_NOT_CONNECT_TO_URL.New(
			errors.Params{
				"url":  url,
				"type": "PUSH",
				"err":  err})

		return nil, err
	}
	return socket, nil
}

func newPacket(msg []byte) [][]byte {
	return [][]byte{[]byte{componentPacket}, msg}
}

func isValidPacket(msg interface{}) bool {
	if msgb, ok := msg.([][]byte); ok {
		if len(msgb[0]) == 1 && msgb[0][0] == componentPacket {
			return true
		}
	}

	return false
}
