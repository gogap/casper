package casper

import (
	"strings"

	"github.com/gogap/errors"

	"github.com/gogap/casper/errorcode"
)

type MessageEvent int32

const (
	MSG_EVENT_PROCESSED MessageEvent = 1
)

type Messenger interface {
	NewMessage(result interface{}) (msg *ComponentMessage, err error)
	ReceiveMessage(msg *ComponentMessage) (err error)
	SendMessage(graphName string, comMsg *ComponentMessage) (msgId string, ch chan *Payload, err error)
	SendToComponent(compMetadata *ComponentMetadata, msg []byte) (total int, err error)
	OnMessageEvent(msgId string, event MessageEvent)
}

type MQChanMessenger struct {
	graphs       Graphs
	compMetadata *ComponentMetadata
	mqCache      map[string]*EndPoint
	requests     map[string]chan *Payload
}

func NewMQChanMessenger(graphs Graphs, compMetadata ComponentMetadata) *MQChanMessenger {

	messenger := new(MQChanMessenger)
	messenger.graphs = graphs
	messenger.requests = make(map[string]chan *Payload)
	messenger.mqCache = make(map[string]*EndPoint)
	messenger.compMetadata = &compMetadata

	return messenger
}

func (p *MQChanMessenger) NewMessage(result interface{}) (msg *ComponentMessage, err error) {
	return NewComponentMessage(p.compMetadata, result)
}

func (p *MQChanMessenger) ReceiveMessage(msg *ComponentMessage) (err error) {
	if ch, exist := p.requests[msg.Id]; !exist {
		bmsg, _ := msg.Serialize()
		err = errorcode.ERR_MESSENGER_REQ_ID_NOT_EXIST.New(
			errors.Params{
				"id":  msg.Id,
				"msg": string(bmsg)})
		return
	} else {
		ch <- msg.Payload
	}
	return nil
}

func (p *MQChanMessenger) SendMessage(graphName string, comMsg *ComponentMessage) (msgId string, ch chan *Payload, err error) {
	// get graph
	graph := p.GetGraph(graphName)
	if graph == nil {
		err = errorcode.ERR_GRAPH_NOT_EXIST.New(errors.Params{"name": graphName})
		return
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
			err = errorcode.ERR_COMPONENT_NOT_EXIST.New(errors.Params{"name": graphName})
			return
		}
		compConf := com.Metadata()
		comMsg.graph = append(comMsg.graph, &compConf)
	}

	// get next com
	nextComp := comMsg.graph[0]

	// new request
	ch = p.addRequest(comMsg.Id)

	// Send Component message
	var message []byte
	if message, err = comMsg.Serialize(); err != nil {
		err = errorcode.ERR_COMPONENT_MSG_SERIALIZE_FAILED.New(
			errors.Params{
				"in":     comMsg.entrance.In,
				"mqType": comMsg.entrance.MQType,
				"err":    err})
		return
	}

	if _, err = p.SendToComponent(nextComp, message); err != nil {
		return
	}

	return comMsg.Id, ch, nil
}

func (p *MQChanMessenger) SendToComponent(compMetadata *ComponentMetadata, msg []byte) (total int, err error) {
	if compMetadata == nil {
		err = errorcode.ERR_COMPONENT_METADATA_IS_NIL.New()
		return
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

func (p *MQChanMessenger) GetGraph(name string) []string {
	if g, ok := p.graphs[name]; ok {
		if len(g) >= 1 {
			return g
		}
	}

	return nil
}
