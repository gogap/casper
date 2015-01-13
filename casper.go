package casper

import (
	"encoding/json"
	"os"
	"strings"

	log "github.com/golang/glog"
	"github.com/cascades-fbp/cascades/runtime"
	"github.com/go-martini/martini"
	. "github.com/gogap/base_component"
	. "github.com/gogap/base_component/utils"
)

var apps map[string]*App

type API struct {
	Name     string
	outPort  []*EndPoint
	dispense string
}

type App struct {
	Name string
	Addr string

	apis     map[string]*API
	inPort   []*EndPoint
	requests map[string]chan *Payload

	martini *martini.ClassicMartini
}

func BuildAppFromConfig(fileName string) {
	var conf struct {
		Apps []struct {
			Name string   `json:"name"`
			Addr string   `json:"addr"`
			In   []string `json:"in"`
			Apis []struct {
				Name     string   `json:"name"`
				Out      []string `json:"out"`
				Dispense string   `json:"dispense"`
			} `json:"apis"`
		} `json:"apps"`

		Loglevel string `json:"loglevel"`
	}

	apps = make(map[string]*App)

	r, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	if err = json.NewDecoder(r).Decode(&conf); err != nil {
		panic(err)
	}

	for i := 0; i < len(conf.Apps); i++ {
		app := NewApp(conf.Apps[i].Name, conf.Apps[i].Addr, conf.Apps[i].In)
		for j := 0; j < len(conf.Apps[i].Apis); j++{
			app.newAPI(conf.Apps[i].Apis[j].Name, conf.Apps[i].Apis[j].Dispense, conf.Apps[i].Apis[j].Out)
		}
	}
}

func GetApp(name string) *App {
	if app, ok := apps[name]; ok {
		return app
	}

	return nil
}

func NewApp(name, addr string, in []string) *App {
	inlen := len(in)
	sname, saddr := strings.TrimSpace(name), strings.TrimSpace(addr)

	if sname == "" || saddr == "" {
		return nil
	}
	if inlen < 1 {
		return nil
	}

	newApp := &App{
		Name:     sname,
		Addr:     saddr,
		apis:     make(map[string]*API),
		inPort:   make([]*EndPoint, inlen),
		requests: make(map[string]chan *Payload),
		martini:  martini.Classic()}

	newApp.martini.Post("/"+sname, handle(newApp))

	// incoming port
	for i := 0; i < inlen; i++ {
		newApp.inPort[i] = &EndPoint{Url: in[i], Socket: nil}
	}

	apps[sname] = newApp

	log.Infoln(newApp)
	
	return newApp
}

func (p *App) newAPI(name, dispense string, out []string) *API {
	outlen := len(out)
	sname, sdispense := strings.TrimSpace(name), strings.TrimSpace(dispense)

	if sname == "" || sdispense == "" {
		return nil
	}
	if outlen < 1 {
		return nil
	}

	newapi := &API{Name: sname, outPort: make([]*EndPoint, outlen), dispense: sdispense}

	for i := 0; i < outlen; i++ {
		newapi.outPort[i] = &EndPoint{Url: out[i], Socket: nil}
	}

	p.apis[sname] = newapi
	
	return newapi
}

func (p *API) run() error {
	var err error
	for i := 0; i < len(p.outPort); i++ {
		p.outPort[i].Socket, err = CreateOutputPort(p.outPort[i].Url)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *App) Run() {
	recver := func(port *EndPoint) {
		for {
			ip, err := port.Socket.RecvMessageBytes(0)
			if err != nil {
				log.Errorln(p.Name, port.Url, "Error receiving message:", err.Error())
				continue
			}
			if !runtime.IsValidIP(ip) {
				log.Errorln(p.Name, port.Url, "Received invalid IP")
				continue
			}
			log.Infoln("recv:", string(ip[1]))
			
			msg := new(ComponentMessage)
			err = msg.FromJson(ip[1])
			if err != nil {
				log.Errorln(p.Name, port.Url, "Format msg error", string(ip[1]))
				continue
			}

			log.Infoln("recvmsg:", msg)

			ch := p.GetRequest(msg.ID)
			if ch == nil {
				log.Errorln(p.Name, port.Url, "404", msg)
				continue
			}

			ch <- msg.Payload
		}
	}

	// run all api
	for _, api := range p.apis {
		err := api.run()
		if err != nil {
			panic(err)
		}
	}

	// Process incoming message forever
	for i := 0; i < len(p.inPort); i++ {
		var err error
		p.inPort[i].Socket, err = CreateInputPort(p.inPort[i].Url)
		if err != nil {
			panic(err)
		}

		go recver(p.inPort[i])
	}

	// Web server goroutine
	p.martini.RunOnAddr(p.Addr)
}

func (p *App) GetApi(name string) *API {
	if api, ok := p.apis[name]; ok {
		return api
	}

	return nil
}

func (p *App) AddRequest(reqid string) (ch chan *Payload) {
	sreqid := strings.TrimSpace(reqid)
	if sreqid == "" {
		return nil
	}

	ch = make(chan *Payload)
	p.requests[sreqid] = ch

	return
}

func (p *App) GetRequest(reqid string) chan *Payload {
	if ch, ok := p.requests[reqid]; ok {
		return ch
	}

	return nil
}

func (p *App) DelRequest(reqid string) {
	delete(p.requests, reqid)
}
