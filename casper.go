package casper

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/cascades-fbp/cascades/runtime"
	"github.com/go-martini/martini"
	"github.com/gogap/base_component"
	"github.com/gogap/pam"
	zmq "github.com/pebbe/zmq4"

	"github.com/gogap/casper/utils"
)

type API struct {
	Name      string
	Endpoints MultiEndpoints
}

func NewAPI(apiName string, endpoints ...MultiEndpoints) (api *API, err error) {
	newapi := new(API)
	//TODO set endpoint
	api = newapi
	return
}

type App struct {
	name string
	host string
	port int32

	inputEndpoint   string
	optionsEndpoint string
	apis            map[string]API

	inChan  chan base_component.ComponentMessage
	outChan chan HandlerRequest

	context             *zmq.Context
	optionsPort, inPort *zmq.Socket

	martini *martini.ClassicMartini
	mux     *pam.PostAPIMux

	isRunning bool
}

func NewApp(appName string, apis ...API) (app *App, err error) {
	newApp := new(App)
	newApp.martini = martini.Classic()
	newApp.mux = pam.New(appName)
	newApp.martini.Post("/"+appName, newApp.mux)
	//TODO set apis
	app = newApp
	return
}

func (p *App) Name() string {
	return p.name
}

func (p *App) Martini() *martini.ClassicMartini {
	return p.martini
}

func (p *App) RegisterAPI(apis ...API) (err error) {
	if p.isRunning {
		return
	}

	for _, api := range apis {
		p.mux.Post(api.Name, handler(p.outChan))
	}
	//TODO put to map
	return
}

func (p *App) openPorts() {
	var err error = nil
	p.context, err = zmq.NewContext()
	if err != nil {
		panic(err)
	}

	p.optionsPort, err = utils.CreateInputPort(p.optionsEndpoint)
	if err != nil {
		panic(err)
	}

	p.inPort, err = utils.CreateInputPort(p.inputEndpoint)
	if err != nil {
		panic(err)
	}
}

func (p *App) closePorts() {
	p.optionsPort.Close()
	if p.inPort != nil {
		p.inPort.Close()
	}
}

func (p *App) run_apis(endpoint string) {
	var outPort *zmq.Socket
	var err error = nil
	if outPort, err = utils.CreateOutputPort(endpoint); err != nil {
		log.Panic(err)
		panic(err)
	}
	defer outPort.Close()

	// Map of uuid to requests
	dataMap := make(map[string]chan base_component.ComponentMessage)

	// Start listening in/out channels
	for {
		select {
		case data := <-p.outChan:
			dataMap[data.Request.Id] = data.ResponseChan
			if ip, e := utils.ComponentMessage2IP(data.Request); e != nil {
				log.Error(e)
				delete(dataMap, data.Request.Id)
				continue
			} else {
				outPort.SendMessage(ip, 0)
			}
		case resp := <-p.inChan:
			if respChan, ok := dataMap[resp.Id]; ok {
				log.Debug("Resolved channel for response", resp.Id)
				respChan <- resp
				delete(dataMap, resp.Id)
				continue
			}
			log.Warning("Didn't find request handler mapping for a given ID", resp.Id)
		}
	}
}

func (p *App) Run() {
	addr := fmt.Sprintf("%s:%d", p.host, p.port)

	for _, api := range p.apis {
		if endpoint, e := api.Endpoints.GetOne(); e != nil {
			log.Panic(e)
			panic(e)
		} else {
			go p.run_apis(endpoint)
		}
	}

	// Web server goroutine
	go func() {
		p.martini.RunOnAddr(addr)
	}()

	// Process incoming message forever
	for {
		ip, err := p.inPort.RecvMessageBytes(0)
		if err != nil {
			log.Debug("Error receiving message:", err.Error())
			continue
		}
		if !runtime.IsValidIP(ip) {
			log.Error("Received invalid IP")
			continue
		}

		msg, err := utils.IP2ComponentMessage(ip)
		if err != nil {
			log.Debug("Error converting IP to response: %s", err.Error())
			continue
		}
		p.inChan <- *msg
	}
}
