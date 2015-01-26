package casper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	log "github.com/golang/glog"
)

var apps map[string]*App = make(map[string]*App)

var entrancefactory EntranceFactory = NewDefaultEntranceFactory()

type HttpResponse struct {
	Code    uint64      `json:"code"`
	Message string      `json:"message,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

type Graphs map[string][]string

type App struct {
	Component
	Entrance

	// graphs   Graphs
	// requests map[string]chan *Payload

	messenger Messenger
}

type CasperConfigs struct {
	Apps       []AppConfig       `json:"apps"`
	Components []ComponentConfig `json:"components"`
}

type AppConfig struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	In          string              `json:"in"`
	MQType      string              `json:"mq_type"`
	Entrance    EntranceOptions     `json:"entrance"`
	Graphs      map[string][]string `json:"graphs"`
}

func (p *AppConfig) ComponentConfig() ComponentConfig {
	return ComponentConfig{
		Name:        p.Name,
		Description: p.Description,
		In:          p.In,
		MQType:      p.MQType}
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
	newApp := &App{}

	compConf := appConf.ComponentConfig()
	compMeta := compConf.Metadata()

	appMessenger := NewMQChanMessenger(appConf.Graphs, compMeta)
	appEntrance := entrancefactory.NewEntrance(appMessenger, appConf.Entrance.Type, appConf.Entrance.Options)

	if appComponent, e := NewComponentWithMessenger(compConf, appMessenger); e != nil {
		panic(e)
	} else {
		newApp.Component = *appComponent
	}

	newApp.messenger = appMessenger
	newApp.Entrance = appEntrance

	app = newApp

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
	if err := p.Component.Run(); err != nil {
		panic(err)
	}
	if err := p.Entrance.Run(); err != nil {
		panic(err)
	}
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
