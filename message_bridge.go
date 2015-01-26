package casper

import (
	"fmt"
	"strings"

	log "github.com/golang/glog"
)

type MessageEvent int32

const (
	MSG_EVENT_PROCESSED MessageEvent = 1
)

type Messenger interface {
	ReceiveMessage(msg *ComponentMessage) (err error)
	SendMessage(graphName string, comMsg *ComponentMessage) (msgId string, ch chan *Payload, err error)
	SendToComponent(compMetadata *ComponentMetadata, msg []byte) (total int, err error)
	OnMessageEvent(msgId string, event MessageEvent)
}

type MQChanMessenger struct {
	graphs       Graphs
	compMetadata *ComponentMetadata
	//components []ComponentConfig
	mqCache  map[string]*EndPoint
	requests map[string]chan *Payload
}

func NewMQChanMessenger(graphs Graphs, compMetadata ComponentMetadata) *MQChanMessenger {

	messenger := new(MQChanMessenger)
	messenger.graphs = graphs
	messenger.requests = make(map[string]chan *Payload)
	messenger.mqCache = make(map[string]*EndPoint)
	messenger.compMetadata = &compMetadata

	return messenger
}

func (p *MQChanMessenger) ReceiveMessage(msg *ComponentMessage) (err error) {
	id := msg.Id
	ch := p.getRequest(id)
	if ch == nil {
		bmsg, _ := msg.Serialize()
		return fmt.Errorf("No such request", id, string(bmsg))
	}

	ch <- msg.Payload

	return nil
}

func (p *MQChanMessenger) SendMessage(graphName string, comMsg *ComponentMessage) (msgId string, ch chan *Payload, err error) {
	// get graph
	graph := p.GetGraph(graphName)
	if graph == nil {
		log.Errorln("No such graph named: ", graphName, p.graphs)
		return "", nil, fmt.Errorf("No such graph named: %s", graphName)
	}

	comMsg.entrance = p.compMetadata

	// build graph
	for i := 0; i < len(graph); i++ {
		if i == 0 && graph[0] == "self" {
			comMsg.graph = append(comMsg.graph, p.compMetadata)
			continue
		}
		com := GetComponentByName(graph[i])
		if com == nil {
			log.Errorln("No such component named: ", graph[i])
			return "", nil, fmt.Errorf("No such component named: ", graph[i])
		}
		compConf := com.Metadata()
		comMsg.graph = append(comMsg.graph, &compConf)
	}
	log.Infoln("msg's graph:", comMsg.graph)

	// get next com
	nextComp := comMsg.graph[0]

	// new request
	ch = p.addRequest(comMsg.Id)

	// Send Component message
	var message []byte
	if msg, e := comMsg.Serialize(); e != nil {
		log.Errorf("Serialize component message error, error is: %s", e)
	} else {
		message = msg
	}

	p.SendToComponent(nextComp, message)

	return comMsg.Id, ch, nil
}

func (p *MQChanMessenger) SendToComponent(compMetadata *ComponentMetadata, msg []byte) (total int, err error) {
	if compMetadata == nil {
		return 0, fmt.Errorf("component metadata is nil")
	}

	if _, ok := p.mqCache[compMetadata.In]; ok == false {
		mqtmp, err := NewMQ(compMetadata)
		if err != nil {
			return 0, err
		}
		p.mqCache[compMetadata.In] = &EndPoint{
			ComponentMetadata: ComponentMetadata{
				In:     compMetadata.In,
				MQType: compMetadata.MQType},
			MessageQueue: mqtmp}
	}

	//log.Infoln(p.Name, "SendToComponent:", in, string(msg))
	total, err = p.mqCache[compMetadata.In].SendToNext(msg)
	return
}

func (p *MQChanMessenger) OnMessageEvent(msgId string, event MessageEvent) {
	switch event {
	case MSG_EVENT_PROCESSED:
		{
			delete(p.requests, msgId)
		}
	}
	return
}

func (p *MQChanMessenger) addRequest(msgId string) (ch chan *Payload) {
	strMsgId := strings.TrimSpace(msgId)
	if strMsgId == "" {
		return nil
	}

	ch = make(chan *Payload)
	p.requests[strMsgId] = ch

	return
}

func (p *MQChanMessenger) getRequest(msgId string) chan *Payload {
	if ch, ok := p.requests[msgId]; ok {
		return ch
	}
	return nil
}

func (p *MQChanMessenger) GetGraph(name string) []string {
	if g, ok := p.graphs[name]; ok {
		if len(g) >= 1 {
			return g
		}
	}

	return nil
}
