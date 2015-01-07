package casper

import (
	"encoding/json"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
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
	logger  *log.Logger
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
				Dospense string   `json:"dispense"`
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

	}
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
		martini:  martini.Classic(),
		logger:   log.New()}

	newApp.martini.Post("/"+sname, handle(newApp))

	// set incoming port
	for i := 0; i < inlen; i++ {
		newApp.inPort[i] = &EndPoint{Url: in[i], Socket: nil}
	}

	apps[sname] = newApp

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
		ip, err := port.Socket.RecvMessageBytes(0)
		if err != nil {
			p.logger.Errorln(p.Name, port.Url, "Error receiving message:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) {
			p.logger.Errorln(p.Name, port.Url, "Received invalid IP")
			continue
		}

		msg := new(ComponentMessage)
		err = msg.FromJson(ip[1])
		if err != nil {
			p.logger.Errorln(p.Name, port.Url, "Format msg error", string(ip[1]))
			continue
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


func (p *App) GetApi(name string) *API{
	if api, ok := p.apis[name]; ok {
		return api
	}
	
	return nil
}

func (p *App) AddRequest(reqid string) (ch chan *Payload){
	sreqid := strings.TrimSpace(reqid)
	if sreqid == "" {
		return nil
	}
	
	ch := make(chan *Payload)
	p.requests[sreqid] = ch

	return ch
}

func (p *App) DelRequest(reqid string) (ch chan *Payload){
	delete(p.requests, reqid)
}
