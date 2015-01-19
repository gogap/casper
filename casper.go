package casper

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/golang/glog"
)

const (
	REQ_TIMEOUT = time.Duration(15) * time.Second
	REQ_X_API   = "X-API"
)

var apps map[string]*App = make(map[string]*App)

type HttpResponse struct {
	Code    uint64      `json:"code"`
	Message string      `json:"message,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

// 服务
type App struct {
	Component
	entrance

	addr     string
	apptype  string
	graphs   map[string][]string
	requests map[string]chan *Payload
}

func BuildAppFromConfigs(filePaths []string) {
	for _, filePath := range filePaths {
		BuildAppFromConfig(filePath)
	}
}

func BuildAppFromConfig(filePath string) {
	var conf struct {
		Apps []struct {
			Name        string              `json:"name"`
			Description string              `json:"description"`
			Entrace     string              `json:"entrace"`
			Addr        string              `json:"addr"`
			Intype      string              `json:"in_type"`
			Inaddr      string              `json:"in_addr"`
			Graphs      map[string][]string `json:"graphs"`
		} `json:"apps"`
		Components []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Type        string `json:"type"`
			In          string `json:"in"`
		} `json:"components"`
	}

	r, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	if err = json.NewDecoder(r).Decode(&conf); err != nil {
		panic(err)
	}

	for _, compConf := range conf.Components {
		_, err := NewComponent(compConf.Name, compConf.Description, compConf.Type, compConf.In)
		if err != nil {
			panic(err)
		}
	}

	for _, appConf := range conf.Apps {
		_, err := NewApp(appConf.Name,
			appConf.Description,
			appConf.Entrace,
			appConf.Addr,
			appConf.Intype,
			appConf.Inaddr,
			appConf.Graphs)
		if err != nil {
			panic(err)
		}
	}
}

func NewApp(name, description, apptype, addr, intype, inaddr string, graphs map[string][]string) (app *App, err error) {
	sname, stype, saddr := strings.TrimSpace(name), strings.TrimSpace(apptype), strings.TrimSpace(addr)
	sintype, sinaddr := strings.TrimSpace(intype), strings.TrimSpace(inaddr)
	if sname == "" {
		return nil, fmt.Errorf("App's name empty error")
	}
	if stype == "" {
		return nil, fmt.Errorf("App's type empty error")
	}
	if saddr == "" {
		return nil, fmt.Errorf("App's addr empty error")
	}
	if sintype == "" {
		return nil, fmt.Errorf("App's intype empty error")
	}
	if sinaddr == "" {
		return nil, fmt.Errorf("App's inaddr empty error")
	}

	app = &App{
		Component: Component{
			Name:        sname,
			Description: description,
			in:          EndPoint{Url: sinaddr, MQType: sintype, mq: nil},
			app:         nil,
			outs:        make(map[string]*EndPoint),
			handler:     nil},
		entrance: nil,
		addr:     saddr,
		apptype:  stype,
		graphs:   graphs,
		requests: make(map[string]chan *Payload)}

	app.app = app
	app.entrance, err = NewEntrance(app.apptype)
	if err != nil {
		return
	}

	log.Infoln(app)
	apps[app.Name] = app

	return
}

func GetAppByName(name string) *App {
	if app, ok := apps[name]; ok {
		return app
	}

	return nil
}

func (p *App) Run() {
	// 验证graph
	for k, v := range p.graphs {
		for i := 0; i < len(v); i++ {
			if GetComponentByName(v[i]) == nil {
				panic(fmt.Sprintf("There is a unknown component name's %s in graph %s", v[i], k))
			}
		}
	}

	p.Component.Run()
	p.entrance.StartService(p, p.addr)
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
		log.Errorln("No such graph named: ", graphName)
		return "", nil, fmt.Errorf("No such graph named: %s", graphName)
	}

	// get com
	nextCom := GetComponentByName(graph[0])
	if nextCom == nil {
		log.Errorln("No such component named: ", graph[0])
		return "", nil, fmt.Errorf("No such component named: ", graph[0])
	}

	comsg.entrance = p.in.Url

	// build graph
	for i := 0; i < len(graph); i++ {
		com := GetComponentByName(graph[i])
		if com == nil {
			log.Errorln("No such component named: ", graph[i])
			return "", nil, fmt.Errorf("No such component named: ", graph[i])
		}
		comsg.graph = append(comsg.graph, com.in.Url)
	}
	log.Infoln("msg's graph:", comsg.graph)

	// new request
	ch = p.addRequest(comsg.ID)

	// Send Component message
	p.sendToNext(nextCom.in.Url, comsg)

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
