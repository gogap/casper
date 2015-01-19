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
	Url    string
	MQType string
	mq
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

func BuildComFromConfig(fileName string) {
	var conf struct {
		Components []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Type        string `json:"type"`
			In          string `json:"in"`
		} `json:"components"`
	}

	r, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	if err = json.NewDecoder(r).Decode(&conf); err != nil {
		panic(err)
	}

	for i := 0; i < len(conf.Components); i++ {
		_, err := NewComponent(conf.Components[i].Name, conf.Components[i].Description, conf.Components[i].Type, conf.Components[i].In)
		if err != nil {
			panic(err)
		}
	}
}

func NewComponent(name, description, mqtype, in string) (*Component, error) {
	sname, smqtype, sin := strings.TrimSpace(name), strings.TrimSpace(mqtype), strings.TrimSpace(in)
	if sname == "" {
		return nil, fmt.Errorf("Component's name empty ERROR")
	}
	if sin == "" {
		return nil, fmt.Errorf("Component's addr empty ERROR")
	}
	if smqtype == "" {
		return nil, fmt.Errorf("Component's mq typpe empty ERROR")
	}

	com := &Component{
		Name:        sname,
		Description: description,
		in:          EndPoint{Url: sin, MQType: smqtype, mq: nil},
		app:         nil,
		outs:        make(map[string]*EndPoint),
		handler:     nil}

	components[com.Name] = com

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

func (p *Component) SetHandler(handler ComponentHandler) {
	if handler == nil {
		panic("handler could not be nil")
	}
	p.handler = handler
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
	log.Infoln("Component Run...", p.Name)

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
		}

		// call worker
		var ret interface{}
		var err error
		if p.handler != nil {
			log.Infoln(p.Name, "Call handler")
			ret, err = p.handler(comsg.Payload)
			if err != nil {
				// 业务处理错误, 发给入口
				log.Errorln("worker error, send to entrance:", smsg)
				if errors.IsErrCode(err) == false {
					comsg.Payload.Code = 500
					comsg.Payload.Message = err.Error()
				} else {
					comsg.Payload.Code = err.(errors.ErrCode).Code()
					comsg.Payload.Message = err.(errors.ErrCode).Error()
				}
				next = comsg.entrance
				comsg.Payload.Result = nil
			} else {
				log.Infoln(p.Name, "Call handler ok")
				comsg.Payload.Result = ret
			}
		}

		// 正常发到下一站
		log.Infoln("sendToNext:", smsg)
		msg, _ := comsg.Serialize()
		if _, err = p.sendToNext(next, msg); err != nil {
			log.Errorf(p.Name, "sendToNext error: ", smsg)
		}
	} else if next == "" {
		// 消息流出错了或是已经走到了入口
		msg, _ := comsg.Serialize()
		if p.app != nil {
			log.Infoln(p.Name, "Msg to entrance:", smsg)
			if err := p.app.recvMsg(comsg); err != nil {
				log.Errorln(p.Name, "msg to entrance err:", err.Error())
			}
		} else {
			// send to entrance
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
