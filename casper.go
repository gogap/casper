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

	apptype string
	graphs  map[string][]string

	requests map[string]chan *Payload
	face     faceI
}

func BuildAppFromConfig(fileName string) {
	var conf struct {
		Apps []struct {
			Name        string              `json:"name"`
			Description string              `json:"description"`
			Type        string              `json:"type"`
			Addr        string              `json:"addr"`
			Intype      string              `json:"intype"`
			Inaddr      string              `json:"inaddr"`
			Graphs      map[string][]string `json:"graphs"`
		} `json:"apps"`
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

	for i := 0; i < len(conf.Apps); i++ {
		_, err := NewApp(conf.Apps[i].Name,
			conf.Apps[i].Description,
			conf.Apps[i].Type,
			conf.Apps[i].Addr,
			conf.Apps[i].Intype,
			conf.Apps[i].Inaddr,
			conf.Apps[i].Graphs)
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
		apptype:  stype,
		graphs:   graphs,
		requests: make(map[string]chan *Payload),
		face:     nil}

	app.app = app
	app.face, err = NewFace(app.apptype)
	if err != nil {
		return
	}
	app.face.SetPara("addr", saddr)
	
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
	// 难证graph
	
	p.Component.Run()

	if p.face != nil {
		p.face.Run(p) // Run user face service
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
	fmt.Println(p.requests)
	
	id := msg.ID
	ch := p.getRequest(id)
	if ch == nil {
		bmsg, _ := msg.Serialize()
		return fmt.Errorf("No such request", id, string(bmsg))
	}

	ch <- msg.Payload
	
	return nil
}

func (p *App) sendMsg(graphName string, msg []byte) (id string, ch chan *Payload, err error) {
	// get graph
	graph := p.GetGraph(graphName)
	if graph == nil {
		log.Errorln("No such graph named: ", graphName)
		return "", nil, fmt.Errorf("No such graph named:", graphName)
	}

	// get com
	nextCom := GetComponentByName(graph[0])
	if nextCom == nil {
		log.Errorln("No such component named: ", graph[0])
		return "", nil, fmt.Errorf("No such component named: ", graph[0])
	}

	// Componet message
	coMsg, _ := NewComponentMessage(p.in.Url)
	coMsg.Payload.SetContext(REQ_X_API, graphName)
	coMsg.Payload.Result = msg

	// build graph
	for i := 0; i < len(graph); i++ {
		com := GetComponentByName(graph[i])
		if com == nil {
			log.Errorln("No such component named: ", graph[i])
			return "", nil, fmt.Errorf("No such component named: ", graph[i])
		}
		coMsg.graph = append(coMsg.graph, com.in.Url)
	}
	log.Infoln("msg's graph:", coMsg.graph)
	
	// new request
	ch = p.addRequest(coMsg.ID)

	// Send Component message
	p.Component.sendToNext(nextCom.in.Url, coMsg)

	return coMsg.ID, ch, nil
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

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// 入口服务的接口
type faceI interface {
	SetPara(key string, val interface{}) // 设置参数
	Run(*App)                            // 开始服务
}

type faceType func() faceI

var faces map[string]faceType = make(map[string]faceType)

func registerFace(name string, one faceType) {
	if one == nil {
		panic("register face nil")
	}
	if _, dup := faces[name]; dup {
		panic("register face duplicate for " + name)
	}
	faces[name] = one
}

func NewFace(typeName string) (faceI, error) {
	newFun, ok := faces[typeName]
	if ok != true {
		return nil, fmt.Errorf("no face types " + typeName)
	}

	return newFun(), nil
}
