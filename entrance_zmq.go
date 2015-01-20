package casper

import (
	"fmt"
	"time"

	log "github.com/golang/glog"
	zmq "github.com/pebbe/zmq4"
)

type zmqEntrance struct {
	socket *zmq.Socket
}

func init() {
	registerEntrance("zmq", NewZmqEntrance)
}

func NewZmqEntrance() entrance {
	return &zmqEntrance{}
}

func (p *zmqEntrance) StartService(app *App, addr string) {
	var err error
	if p.socket, err = zmq.NewSocket(zmq.REP); err != nil {
		panic(err)
	}
	if err = p.socket.Bind(addr); err != nil {
		panic(err)
	}
	log.Infoln("zmqEntrance start at:", addr)

	p.zmqEntranceHandler(app)
}

func (p *zmqEntrance) zmqEntranceHandler(app *App) {
	for {
		msg, err := p.socket.RecvMessageBytes(0)
		if err != nil {
			log.Errorln("recvMessage err:", err.Error())
			p.socket.SendMessage(newPacket([]byte("ERR")))
			continue
		}

		if !isValidPacket(msg) {
			log.Errorln("RecvMessage invalid message.")
			p.socket.SendMessage(newPacket([]byte("ERR")))
			continue
		}

		log.Infoln("recvMessage:", string(msg[1]))

		coMsg, _ := NewComponentMessage("")
		err = coMsg.FromJson(msg[1])
		if err != nil {
			log.Errorln("RecvMessage message fmt error.")
			p.socket.SendMessage(newPacket([]byte("ERR")))
			continue
		}

		log.Infoln("recvComsg:", coMsg)

		apiName, err := coMsg.Payload.GetContextString(REQ_X_API)
		if err != nil {
			log.Errorln("Get message's X-API error.", err.Error())
			p.socket.SendMessage(newPacket([]byte("ERR")))
			continue
		}
		if apiName == "" {
			log.Errorln("Get message's X-API NULL.")
			p.socket.SendMessage(newPacket([]byte("ERR")))
			continue
		}

		// send msg to next
		id, ch, err := app.sendMsg(apiName, coMsg)
		if err != nil {
			log.Errorln("sendMsg err:", coMsg.ID, err.Error())
			p.socket.SendMessage(newPacket([]byte("ERR")))
			continue
		}
		if ch == nil {
			log.Errorln("sendMsg return nil:", coMsg.ID)
			p.socket.SendMessage(newPacket([]byte("ERR")))
			continue
		}
		defer close(ch)
		defer app.delRequest(id)

		// Wait for response from IN port
		log.Infoln("Waiting for response: ", apiName, string(msg[1]))
		var load *Payload
		select {
		case load = <-ch:
			break
		case <-time.Tick(REQ_TIMEOUT):
			p.socket.SendMessage(newPacket([]byte("TIMEOUT")))
			return
		}

		coMsg.Payload = load
		rst, _ := coMsg.Serialize()
		p.socket.SendMessage(newPacket(rst))
	}
}

func zmqSyncCall(endpoint string, request *ComponentMessage) (reply *ComponentMessage, err error) {
	client, err := zmq.NewSocket(zmq.REQ)
	if err != nil {
		return nil, err
	}
	if err := client.Connect(endpoint); err != nil {
		return nil, err
	}
	log.Infoln("zmq connect ok:", endpoint)

	msg, _ := request.Serialize()
	packet := newPacket(msg)
	total, err := client.SendMessage(packet)
	if err != nil {
		return nil, err
	}
	log.Infoln("sendok: ", total, endpoint, string(msg))

	poller := zmq.NewPoller()
	poller.Add(client, zmq.POLLIN)
	polled, err := poller.Poll(REQ_TIMEOUT)
	if err != nil {
		return nil, err
	}

	if len(polled) == 1 {
		ip, err := client.RecvMessageBytes(0)
		if err != nil {
			log.Errorln("recvmsg err:", err.Error())
			return nil, err
		}

		if !isValidPacket(ip) {
			return nil, fmt.Errorf("recv not valid message")
		}
		
		rst := new(ComponentMessage)
		rst.FromJson(ip[1])
		return rst, nil
	}

	return nil, fmt.Errorf("Time out")
}
