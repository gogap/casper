package casper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	log "github.com/golang/glog"
)

var apps map[string]*App = make(map[string]*App)

var entrancefactory EntranceFactory = NewDefaultEntranceFactory()

type HttpResponse struct {
	Code    uint64      `json:"code"`
	Message string      `json:"message,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

type App struct {
	Component
	Entrance

	graphs   map[string][]string
	requests map[string]chan *Payload
}

type CasperConfigs struct {
	Apps       []AppConfig       `json:"apps"`
	Components []ComponentConfig `json:"components"`
}

type AppConfig struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	In          string              `json:"in"`
	InType      string              `json:"in_type"`
	Entrance    EntranceOptions     `json:"entrance"`
	Graphs      map[string][]string `json:"graphs"`
}

func BuildApps(filePaths []string) {
	for _, filePath := range filePaths {
		BuildApp(filePath)
	}
}

func BuildApp(filePath string) {
	conf := CasperConfigs{}

	if bConf, e := ioutil.ReadFile(filePath); e != nil {
		panic(e)
	} else if e := json.Unmarshal(bConf, &conf); e != nil {
		panic(e)
	}

	for _, compConf := range conf.Components {
		if _, e := NewComponent(compConf); e != nil {
			panic(e)
		}
	}

	for _, appConf := range conf.Apps {
		if _, e := NewApp(appConf); e != nil {
			panic(e)
		}
	}
}

func NewApp(appConf AppConfig) (app *App, err error) {
	app = &App{
		Component: Component{
			Name:        appConf.Name,
			Description: appConf.Description,
			in:          EndPoint{Url: appConf.In, MQType: appConf.InType, mq: nil},
			app:         nil,
			outs:        make(map[string]*EndPoint),
			handler:     nil},
		Entrance: nil,
		graphs:   appConf.Graphs,
		requests: make(map[string]chan *Payload)}

	app.app = app
	app.Entrance = entrancefactory.NewEntrance(appConf.Entrance.Type, app, appConf.Entrance.Options)

	apps[app.Name] = app
	log.Infoln("NewApp:", app)
	return
}

func GetAppByName(name string) *App {
	if app, ok := apps[name]; ok {
		return app
	}

	return nil
}

func RegisterEntrances(entrances ...Entrance) {
	for _, entrance := range entrances {
		entrancefactory.RegisterEntrance(entrance)
	}
}

func SetEntranceFactory(factory EntranceFactory) {
	if factory == nil {
		panic("could not set a nil EntranceFactory")
	}

	entrancefactory = factory
}

func (p *App) Run() {
	for k, v := range p.graphs {
		for i := 0; i < len(v); i++ {
			if v[i] == "self" {
				continue
			}
			if GetComponentByName(v[i]) == nil {
				panic(fmt.Sprintf("There is a unknown component name's %s in graph %s", v[i], k))
			}
		}
	}

	if err := p.Component.Run(); err != nil {
		panic(err)
	}
	if err := p.Entrance.Run(); err != nil {
		panic(err)
	}
}

func (p *App) GetGraph(name string) []string {
	if g, ok := p.graphs[name]; ok {
		if len(g) >= 1 {
			return g
		}
	}

	return nil
}

func (p *App) recvMsg(msg *ComponentMessage) error {
	id := msg.ID
	ch := p.getRequest(id)
	if ch == nil {
		bmsg, _ := msg.Serialize()
		return fmt.Errorf("No such request", id, string(bmsg))
	}

	ch <- msg.Payload

	return nil
}

func (p *App) sendMsg(graphName string, comsg *ComponentMessage) (id string, ch chan *Payload, err error) {
	// get graph
	graph := p.GetGraph(graphName)
	if graph == nil {
		log.Errorln("No such graph named: ", graphName, p.graphs)
		return "", nil, fmt.Errorf("No such graph named: %s", graphName)
	}

	comsg.entrance = p.Component.in.Url

	// build graph
	for i := 0; i < len(graph); i++ {
		if i == 0 && graph[0] == "self" {
			comsg.graph = append(comsg.graph, p.Component.in.Url)
			continue
		}

		com := GetComponentByName(graph[i])
		if com == nil {
			log.Errorln("No such component named: ", graph[i])
			return "", nil, fmt.Errorf("No such component named: ", graph[i])
		}
		comsg.graph = append(comsg.graph, com.in.Url)
	}
	log.Infoln("msg's graph:", comsg.graph)

	// get com
	nextCom := comsg.graph[0]

	// new request
	ch = p.addRequest(comsg.ID)

	// Send Component message
	msg, _ := comsg.Serialize()
	p.Component.sendToNext(nextCom, msg)

	return comsg.ID, ch, nil
}

func (p *App) addRequest(reqid string) (ch chan *Payload) {
	sreqid := strings.TrimSpace(reqid)
	if sreqid == "" {
		return nil
	}

	ch = make(chan *Payload)
	p.requests[sreqid] = ch

	return
}

func (p *App) getRequest(reqid string) chan *Payload {
	if ch, ok := p.requests[reqid]; ok {
		return ch
	}

	return nil
}

func (p *App) delRequest(reqid string) {
	delete(p.requests, reqid)
}

func CallService(serviceType, addr string, msg *ComponentMessage) (reply *ComponentMessage, err error) {
	switch serviceType {
	case "zmq":
		{
			return zmqSyncCall(addr, msg)
		}
	case "http":
		{

		}
	}

	return nil, fmt.Errorf("No such serviceType %s", serviceType)
}
