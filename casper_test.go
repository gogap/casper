package casper_test

import (
	"fmt"
	"testing"

	"github.com/gogap/casper"
)

func TestNewApp(t *testing.T) {
	graphs := make(map[string][]string)
	graphs["graph1"] = append(graphs["graph1"], "com1")
	graphs["graph1"] = append(graphs["graph1"], "com2")
	graphs["graph1"] = append(graphs["graph1"], "com3")
	
	app, _ := casper.NewApp("demo", "demo app", "http", "127.0.0.1:8080", "zmq", "tcp://127.0.0.1:5000", graphs)

	fmt.Println(*app)
}


