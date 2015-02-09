/*
 组件
*/
package casper

import (
	"encoding/json"
	"os"

	"github.com/gogap/errors"
	"github.com/gogap/logs"

	"github.com/gogap/casper/errorcode"
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
	logs.Info("load components config file:", fileName)

	if err != nil {
		err = errorcode.ERR_OPENFILE_ERROR.New(errors.Params{"fileName": fileName, "err": err})
		logs.Error(err)
		panic(err)
	}
	defer r.Close()

	if err = json.NewDecoder(r).Decode(&conf); err != nil {
		err = errorcode.ERR_JSON_UNMARSHAL_ERROR.New(errors.Params{"err": err})
		logs.Error(err)
		panic(err)
	}

	for _, compConf := range conf.Components {
		if _, err = NewComponent(compConf); err != nil {
			logs.Error(err)
			panic(err)
		}
	}

}

func NewComponent(conf ComponentConfig) (component *Component, err error) {
	messenger := NewMQChanMessenger(nil, conf.Metadata())
	return NewComponentWithMessenger(conf, messenger)
}

func NewComponentWithMessenger(conf ComponentConfig, messenger Messenger) (component *Component, err error) {
	comp := &Component{
		Name:        conf.Name,
		Description: conf.Description,
		endPoint:    EndPoint{ComponentMetadata: ComponentMetadata{In: conf.In, MQType: conf.MQType}, MessageQueue: nil},
		messenger:   messenger,
		handler:     nil}

	components[comp.Name] = comp

	logs.Pretty("new component:", comp)
	return comp, nil
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
			err := errorcode.ERR_COMPONENT_NOT_EXIST.New(errors.Params{"name": name})
			logs.Error(err)
			panic(err)
		}
	}
}

func (p *Component) SetHandler(handler ComponentHandler) *Component {
	if handler == nil {
		err := errorcode.ERR_COMPONENT_HANDLER_IS_NIL.New()
		logs.Error(err)
		panic(err)
	}
	p.handler = handler
	return p
}

func (p *Component) Run() (err error) {
	SingleInstane("/tmp/" + p.Name + ".pid")
	logs.Info("component running:", p.Name, p.endPoint.In)

	if p.endPoint.MessageQueue, err = NewMQ(&p.endPoint.ComponentMetadata); err != nil {
		return
	}

	err = p.endPoint.Ready()
	if err != nil {
		return
	}

	go p.recvMonitor()

	return nil
}

func (p *Component) recvMonitor() {
	for {
		msg, err := p.endPoint.RecvMessage()
		if err != nil {
			logs.Error(err)
			continue
		}

		strMsg := string(msg)
		logs.Debug(p.Name, "Recv:", strMsg)

		comMsg := new(ComponentMessage)
		if err := comMsg.FromJson(msg); err != nil {
			err = errorcode.ERR_COULD_NOT_PARSE_COMPONENT_MSG.New(
				errors.Params{"in": p.endPoint.In,
					"mqType": p.endPoint.MQType,
					"msg":    strMsg})

			logs.Error(err)
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
			logs.Warn("next is nil. send to entrance:", strMsg)
			next = comMsg.entrance
			comMsg.graph = nil
		}

		// call handler
		var ret interface{}
		var err error
		if p.handler != nil {
			logs.Debug(p.Name, "begin call handler")
			ret, err = p.handler(comMsg.Payload)
			comMsg.Payload.result = nil
			if err != nil {
				// 业务处理错误, 发给入口
				warnErr := errorcode.ERR_HANDLER_RETURN_ERROR.New(errors.Params{"name": p.Name})
				logs.Error(warnErr)

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
				logs.Debug(p.Name, "end call handler")
			}
		}

		// 正常发到下一站
		logs.Debug("begin send to next component:", next.In, next.MQType, strMsg)
		if msg, e := comMsg.Serialize(); e != nil {
			err = errorcode.ERR_COULD_NOT_PARSE_COMPONENT_MSG.New(
				errors.Params{"in": p.endPoint.In,
					"mqType": p.endPoint.MQType,
					"msg":    strMsg,
					"err":    e})
			logs.Error(err)
		} else if _, err = p.messenger.SendToComponent(next, msg); err != nil {
			logs.Error(err)
		}
	} else if next == nil || next.In == "" {
		// 消息流出错了或是已经走到了入口
		msg, _ := comMsg.Serialize()
		if p.messenger != nil {
			// 到入口了, 抛给上层
			logs.Debug(p.Name, "send msg to entrance:", comMsg.entrance.Name, comMsg.entrance.In, comMsg.entrance.MQType)
			if err := p.messenger.ReceiveMessage(comMsg); err != nil {
				logs.Error(err)
			}
		} else {
			// 链路错了， 发给入口
			logs.Debug("msg's next null, send to entrance", strMsg)
			if _, err := p.messenger.SendToComponent(comMsg.entrance, msg); err != nil {
				logs.Error(err)
			}
		}
	} else if next.In != p.endPoint.In {
		// 发给正确的站点
		if msg, err := comMsg.Serialize(); err != nil {
			err = errorcode.ERR_COMPONENT_MSG_SERIALIZE_FAILED.New(
				errors.Params{
					"in":     p.endPoint.In,
					"mqType": p.endPoint.MQType,
					"err":    err})
			logs.Error(err)
		} else if _, err = p.messenger.SendToComponent(next, msg); err != nil {
			logs.Error(err)
		}
	}
}
