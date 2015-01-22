package casper

import (
	"fmt"
	"testing"
)

func TestNewApp(t *testing.T) {
	graphs := make(map[string][]string)
	graphs["graph1"] = append(graphs["graph1"], "com1")
	graphs["graph1"] = append(graphs["graph1"], "com2")
	graphs["graph1"] = append(graphs["graph1"], "com3")
	
	app, _ := NewApp("demo", "demo app", "http", "127.0.0.1:8080", "zmq", "test", "tcp://127.0.0.1:5000", graphs)

	fmt.Println(*app)
}


