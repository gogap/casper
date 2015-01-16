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
	mq     mqI
}

// 组件
type Component struct {
	Name        string
	Description string
	in          EndPoint
	app         *App

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
	p.in.mq, err = NewMq(p.in.MQType)
	if err != nil {
		return
	}
	p.in.mq.SetPara("url", p.in.Url)

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
		log.Infoln("recvmsg:", p.Name, string(msg))
		fmt.Println(string(msg))

		comsg := new(ComponentMessage)
		if err := comsg.FromJson(msg); err != nil {
			log.Errorln("Msg'sformat error:", p.Name, err.Error(), string(msg))
			continue
		}

		// update chain
		comsg.chain = append(comsg.chain, p.in.Url)

		// deal path
		next := comsg.TopGraph()
		if next == p.in.Url {
			next = comsg.PopGraph() // pop to next
			if next == "" {
				log.Warningln("next is nil. next to entrance:", comsg.entrance)
				next = comsg.entrance
			}

			// call worker
			var ret interface{}
			if p.handler != nil {
				ret, err = p.handler(comsg.Payload)
				if err != nil {
					if errors.IsErrCode(err) == false {
						comsg.Payload.Code = 500
					} else {
						comsg.Payload.Code = err.(errors.ErrCode).Code()
						comsg.Payload.Message = err.(errors.ErrCode).Error()
					}
					comsg.Payload.Result = nil

					// send to entrance
					total, err := p.sendToNext(comsg.entrance, comsg)
					log.Errorln("worker error, send to entrance", p.Name, total, string(msg))
					if err != nil {
						log.Errorln("worker error, send to entrance ERR", p.Name, string(msg), err)
					}

					continue
				} else {
					comsg.Payload.Result = ret
				}
			}

			// send to next
			if _, err = p.sendToNext(next, comsg); err != nil {
				log.Errorf(p.Name, string(msg), "send message next component error: ", err)
			}

		} else if next == "" {
			if p.app != nil {
				log.Infoln("recvMsg to app:", p.Name, string(msg))
				if err := p.app.recvMsg(comsg); err != nil {
					log.Errorln("recvmsg to caper err:", p.Name, err.Error())
				}
			} else {
				// send to entrance
				total, err := p.sendToNext(comsg.entrance, comsg)
				if err != nil {
					log.Errorln("msg's path null, send to entrance ERR", p.Name, string(msg))
				}
				log.Errorln("msg's path null, send to entrance", p.Name, total, string(msg))
			}
			continue
		} else if next != p.in.Url {
			// send to real next
			total, err := p.sendToNext(next, comsg)
			if err != nil {
				log.Errorln("msg's path null, send to real next ERR", p.Name, string(msg))
			}
			log.Errorln("msg's path err, send to real next", p.Name, total, string(msg))
			continue
		}
	}
}

func (p *Component) sendToNext(url string, msg *ComponentMessage) (total int, err error) {
	if url == "" {
		return 0, fmt.Errorf("sendTo nil url")
	}

	if _, ok := p.outs[url]; ok == false {
		p.outs[url] = &EndPoint{Url: url, MQType: p.in.MQType, mq: nil}
		p.outs[url].mq, err = NewMq(p.in.MQType) 
		if err != nil {
			return
		}
		p.outs[url].mq.SetPara("url", url)
	}

	var msgb []byte
	msgb, err = msg.Serialize()
	if err != nil {
		return
	}

	log.Infoln("sendToNext:", url, string(msgb))
	total, err = p.outs[url].mq.SendToNext(msgb)

	return
}
