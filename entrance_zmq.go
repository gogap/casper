package casper

import (
	"fmt"
	"time"

	log "github.com/golang/glog"
	zmq "github.com/pebbe/zmq4"

	"github.com/gogap/logs"
)

type EntranceZMQ struct {
	address   string
	app       *App
	socket    *zmq.Socket
	messenger Messenger
}

func init() {
	entrancefactory.RegisterEntrance(new(EntranceZMQ))
}

func (p *EntranceZMQ) Type() string {
	return "zmq"
}

func (p *EntranceZMQ) Init(messenger Messenger, configs EntranceConfig) (err error) {
	if addr, ok := configs.GetConfigString("address"); !ok {
		err = fmt.Errorf("[entrance-%s] get config section of %s failed", p.Type(), "address")
	} else {
		p.address = addr
	}

	if messenger == nil {
		err = fmt.Errorf("[entrance-%s] Messenger is nil", p.Type())
		logs.Info(err)
		return
	} else {
		p.messenger = messenger
	}

	return
}

func (p *EntranceZMQ) Run() error {
	var err error

	if p.socket, err = zmq.NewSocket(zmq.REP); err != nil {
		return err
	}

	if err = p.socket.Bind(p.address); err != nil {
		return err
	}

	logs.Info("entrance", p.Type(), "start:", p.address)
	p.EntranceZMQHandler()

	return nil
}

func (p *EntranceZMQ) EntranceZMQHandler() {
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

		comMsg, _ := NewComponentMessage(nil, nil)
		err = comMsg.FromJson(msg[1])
		if err != nil {
			log.Errorln("RecvMessage message fmt error.")
			p.socket.SendMessage(newPacket([]byte("ERR")))
			continue
		}

		log.Infoln("recvComsg:", comMsg)

		apiName, err := comMsg.Payload.GetContextString(REQ_X_API)
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

		go func(api string, msg *ComponentMessage) {
			// send msg to next
			id, ch, err := p.messenger.SendMessage(api, msg)
			if err != nil {
				log.Errorln("sendMsg err:", msg.Id, err.Error())
				p.socket.SendMessage(newPacket([]byte("ERR")))
				return
			}
			if ch == nil {
				log.Errorln("sendMsg return nil:", msg.Id)
				p.socket.SendMessage(newPacket([]byte("ERR")))
				return
			}

			defer close(ch)
			defer p.messenger.OnMessageEvent(id, MSG_EVENT_PROCESSED)

			// Wait for response from IN port
			log.Infoln("Waiting for response: ", api)
			var load *Payload
			select {
			case load = <-ch:
				break
			case <-time.Tick(REQ_TIMEOUT):
				p.socket.SendMessage(newPacket([]byte("TIMEOUT")))
				return
			}

			comMsg.Payload = load
			rst, _ := comMsg.Serialize()
			p.socket.SendMessage(newPacket(rst))
		}(apiName, comMsg)
	}
}

func zmqSyncCall(endpoint string, request *ComponentMessage) (reply *ComponentMessage, err error) {
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint is nil")
	}
	if request == nil {
		return nil, fmt.Errorf("request is nil")
	}

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
