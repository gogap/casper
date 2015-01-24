/*
 组件
*/
package casper

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gogap/errors"
	log "github.com/golang/glog"
)

var components map[string]*Component = make(map[string]*Component)

// 端点
type EndPoint struct {
	mq
	Url    string
	MQType string
}

// 组件
type Component struct {
	Name        string
	Description string
	in          EndPoint
	app         *App // 当这是一个入口组件...

	outs    map[string]*EndPoint
	handler ComponentHandler
}

type ComponentHandler func(*Payload) (result interface{}, err error)
type ComponentHandlers map[string]ComponentHandler

type ComponentConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	MQType      string `json:"mq_type"`
	In          string `json:"in"`
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
	com := &Component{
		Name:        conf.Name,
		Description: conf.Description,
		in:          EndPoint{Url: conf.In, MQType: conf.MQType, mq: nil},
		app:         nil,
		outs:        make(map[string]*EndPoint),
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

func (p *Component) GetOutPoint(url string) *EndPoint {
	surl := strings.TrimSpace(url)
	if surl == "" {
		return nil
	}

	if _, ok := p.outs[surl]; !ok {
		p.outs[surl] = &EndPoint{Url: surl, MQType: p.in.MQType, mq: nil}
	}

	return p.outs[surl]
}

func (p *Component) Run() (err error) {
	log.Infof("Component Running..... Name:%s, In:%s\n", p.Name, p.in.Url)

	// 创建MQ
	p.in.mq, err = NewMq(p.in.MQType, p.in.Url)
	if err != nil {
		return
	}

	// MQ 准备
	err = p.in.mq.Ready()
	if err != nil {
		return
	}

	// 开始监听
	go p.recvMonitor()

	return nil
}

func (p *Component) recvMonitor() {
	for {
		msg, err := p.in.mq.RecvMessage()
		if err != nil {
			log.Errorln(p.Name, "Error receiving message:", err.Error())
			continue
		}
		log.Infoln(p.Name, "Recv:", string(msg))

		comsg := new(ComponentMessage)
		if err := comsg.FromJson(msg); err != nil {
			log.Errorln(p.Name, "Msg's format error:", err.Error(), string(msg))
			continue
		}

		go p.SendMsg(comsg)
	}
}

func (p *Component) SendMsg(comsg *ComponentMessage) {
	// 就是打日志用的
	msg, _ := comsg.Serialize()
	smsg := string(msg)

	// 更新调用链
	comsg.chain = append(comsg.chain, p.in.Url)

	// deal path
	next := comsg.TopGraph()
	if next == p.in.Url {
		// 正常流程
		next = comsg.PopGraph()
		if next == "" {
			log.Warningln("next is nil. send to entrance:", smsg)
			next = comsg.entrance
			comsg.graph = nil
		}

		// call worker
		var ret interface{}
		var err error
		if p.handler != nil {
			log.Infoln(p.Name, "Call handler")
			ret, err = p.handler(comsg.Payload)
			comsg.Payload.result = nil
			if err != nil {
				// 业务处理错误, 发给入口
				log.Warningln("worker error, send to entrance:", smsg, err.Error())
				if errors.IsErrCode(err) == false {
					comsg.Payload.Code = 500
					comsg.Payload.Message = err.Error()
				} else {
					comsg.Payload.Code = err.(errors.ErrCode).Code()
					comsg.Payload.Message = err.(errors.ErrCode).Error()
				}
				next = comsg.entrance
				comsg.graph = nil
			} else {
				if ret != nil {
					err = comsg.Payload.setResult(ret)
					if err != nil {
						log.Errorln("work result Marshal:", err.Error(), ret)
						comsg.Payload.Code = 500
						comsg.Payload.Message = fmt.Sprintf("%v. %v", p.Name, err.Error())
						next = comsg.entrance
						comsg.graph = nil
					}
				}
				log.Infoln(p.Name, "Call handler ok")
			}
		}

		// 正常发到下一站
		log.Infoln("sendToNext:", next, smsg)
		msg, _ := comsg.Serialize()
		if _, err = p.sendToNext(next, msg); err != nil {
			log.Errorf(p.Name, "sendToNext error: ", smsg)
		}
	} else if next == "" {
		// 消息流出错了或是已经走到了入口
		msg, _ := comsg.Serialize()
		if p.app != nil {
			// 到入口了, 抛给上层
			log.Infoln(p.Name, "Msg to entrance:", smsg)
			if err := p.app.recvMsg(comsg); err != nil {
				log.Errorln(p.Name, "msg to entrance err:", err.Error())
			}
		} else {
			// 链路错了， 发给入口
			log.Errorln("msg's next null, send to entrance", smsg)
			_, err := p.sendToNext(comsg.entrance, msg)
			if err != nil {
				log.Errorln(p.Name, "msg's next null, send to entrance ERR", smsg)
			}
		}
	} else if next != p.in.Url {
		// 发给正确的站点
		msg, _ := comsg.Serialize()
		log.Warningln(p.Name, "msg's real next is:", next)
		_, err := p.sendToNext(next, msg)
		if err != nil {
			log.Errorln(p.Name, "send to real next ERR: ", string(msg))
		}
	}
}

func (p *Component) sendToNext(url string, msg []byte) (total int, err error) {
	if url == "" {
		return 0, fmt.Errorf("sendTo nil url")
	}

	if _, ok := p.outs[url]; ok == false {
		mqtmp, err := NewMq(p.in.MQType, url)
		if err != nil {
			return 0, err
		}
		p.outs[url] = &EndPoint{Url: url, MQType: p.in.MQType, mq: mqtmp}
	}

	log.Infoln(p.Name, "sendToNext:", url, string(msg))
	total, err = p.outs[url].mq.SendToNext(msg)

	return
}
