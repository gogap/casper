/*
 组件
*/
package casper

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gogap/errors"
	log "github.com/golang/glog"
)

var components map[string]*Component = make(map[string]*Component)

// 端点
type EndPoint struct {
	MessageQueue
	ComponentMetadata
}

// 组件
type Component struct {
	Name        string
	Description string
	endPoint    EndPoint
	messenger   Messenger

	handler ComponentHandler
}

func (p *Component) Metadata() ComponentMetadata {
	return ComponentMetadata{
		Name:   p.Name,
		In:     p.endPoint.In,
		MQType: p.endPoint.MQType}
}

func (p *Component) GetComponentConfig() ComponentConfig {
	return ComponentConfig{
		Name:        p.Name,
		Description: p.Description,
		In:          p.endPoint.In,
		MQType:      p.endPoint.MQType}
}

type ComponentHandler func(*Payload) (result interface{}, err error)
type ComponentHandlers map[string]ComponentHandler

type ComponentConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	MQType      string `json:"mq_type"`
	In          string `json:"in"`
}

func (p *ComponentConfig) Metadata() ComponentMetadata {
	return ComponentMetadata{
		Name:   p.Name,
		In:     p.In,
		MQType: p.MQType}
}

func BuildComponent(fileName string) {
	var conf struct {
		Components []ComponentConfig `json:"components"`
	}

	r, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	if err = json.NewDecoder(r).Decode(&conf); err != nil {
		panic(err)
	}

	for _, compConf := range conf.Components {
		if _, err := NewComponent(compConf); err != nil {
			panic(err)
		}
	}

}

func NewComponent(conf ComponentConfig) (component *Component, err error) {
	messenger := NewMQChanMessenger(nil, conf.Metadata())
	com := &Component{
		Name:        conf.Name,
		Description: conf.Description,
		endPoint:    EndPoint{ComponentMetadata: ComponentMetadata{In: conf.In, MQType: conf.MQType}, MessageQueue: nil},
		messenger:   messenger,
		handler:     nil}

	components[com.Name] = com

	log.Infoln(com)
	return com, nil
}

func NewComponentWithMessenger(conf ComponentConfig, messenger Messenger) (component *Component, err error) {
	com := &Component{
		Name:        conf.Name,
		Description: conf.Description,
		endPoint:    EndPoint{ComponentMetadata: ComponentMetadata{In: conf.In, MQType: conf.MQType}, MessageQueue: nil},
		messenger:   messenger,
		handler:     nil}

	components[com.Name] = com

	log.Infoln(com)
	return com, nil
}

func GetComponentByName(name string) *Component {
	if component, ok := components[name]; ok {
		return component
	}
	return nil
}

func SetHandlers(handlers ComponentHandlers) {
	for name, handler := range handlers {
		if component := GetComponentByName(name); component != nil {
			component.SetHandler(handler)
		} else {
			panic(fmt.Errorf("component of %s not exist", name))
		}
	}
}

func (p *Component) SetHandler(handler ComponentHandler) *Component {
	if handler == nil {
		panic("handler could not be nil")
	}
	p.handler = handler

	return p
}

func (p *Component) Run() (err error) {
	log.Infof("[Component-%s] Run at:%s\n", p.Name, p.endPoint.In)

	// 创建MQ
	p.endPoint.MessageQueue, err = NewMQ(&p.endPoint.ComponentMetadata)
	if err != nil {
		return
	}

	// MQ 准备
	err = p.endPoint.Ready()
	if err != nil {
		return
	}

	// 开始监听
	go p.recvMonitor()

	return nil
}

func (p *Component) recvMonitor() {
	for {
		msg, err := p.endPoint.RecvMessage()
		if err != nil {
			log.Errorln(p.Name, "Error receiving message:", err.Error())
			continue
		}
		log.Infoln(p.Name, "Recv:", string(msg))

		comMsg := new(ComponentMessage)
		if err := comMsg.FromJson(msg); err != nil {
			log.Errorln(p.Name, "Msg's format error:", err.Error(), string(msg))
			continue
		}

		go p.SendMsg(comMsg)
	}
}

func (p *Component) SendMsg(comMsg *ComponentMessage) {
	// 就是打日志用的
	msg, _ := comMsg.Serialize()
	strMsg := string(msg)

	// 更新调用链
	comMsg.chain = append(comMsg.chain, p.endPoint.In)

	// deal path
	next := comMsg.TopGraph()

	if next != nil && (next.In == p.endPoint.In) {
		// 正常流程
		next = comMsg.PopGraph()
		if next == nil || next.Name == "" {
			log.Warningln("next is nil. send to entrance:", strMsg)
			next = comMsg.entrance
			comMsg.graph = nil
		}

		// call worker
		var ret interface{}
		var err error
		if p.handler != nil {
			log.Infoln(p.Name, "Call handler")
			ret, err = p.handler(comMsg.Payload)
			comMsg.Payload.result = nil
			if err != nil {
				// 业务处理错误, 发给入口
				log.Warningln("worker error, send to entrance:", strMsg, err.Error())
				if errors.IsErrCode(err) == false {
					comMsg.Payload.Code = 500
					comMsg.Payload.Message = err.Error()
				} else {
					comMsg.Payload.Code = err.(errors.ErrCode).Code()
					comMsg.Payload.Message = err.(errors.ErrCode).Error()
				}
				next = comMsg.entrance
				comMsg.graph = nil
			} else {
				comMsg.Payload.result = ret
				log.Infoln(p.Name, "Call handler ok")
			}
		}

		// 正常发到下一站
		log.Infoln("sendToNext:", next, strMsg)
		msg, _ := comMsg.Serialize()
		if _, err = p.messenger.SendToComponent(next, msg); err != nil {
			log.Errorf("SendToComponent %s error, %s", p.Name, err)
		}
	} else if next == nil || next.In == "" {
		// 消息流出错了或是已经走到了入口
		msg, _ := comMsg.Serialize()
		if p.messenger != nil {
			// 到入口了, 抛给上层
			log.Infoln(p.Name, "Msg to entrance:", strMsg)
			if err := p.messenger.ReceiveMessage(comMsg); err != nil {
				log.Errorln(p.Name, "msg to entrance err:", err.Error())
			}
		} else {
			// 链路错了， 发给入口
			log.Errorln("msg's next null, send to entrance", strMsg)
			_, err := p.messenger.SendToComponent(comMsg.entrance, msg)
			if err != nil {
				log.Errorln(p.Name, "msg's next null, send to entrance ERR", strMsg)
			}
		}
	} else if next.In != p.endPoint.In {
		// 发给正确的站点
		msg, _ := comMsg.Serialize()
		log.Warningln(p.Name, "msg's real next is:", next)
		_, err := p.messenger.SendToComponent(next, msg)
		if err != nil {
			log.Errorln(p.Name, "send to real next ERR: ", string(msg))
		}
	}
}
