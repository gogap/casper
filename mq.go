package casper

import (
	"fmt"
)

var mqs map[string]mqType = make(map[string]mqType)

// 消息接口
type mq interface {
	Ready() error                   // 初始化
	RecvMessage() ([]byte, error)   // 读一条消息
	SendToNext([]byte) (int, error) // 发送一条消息到下一节点
}

type mqType func(string) mq

func registerMq(name string, one mqType) {
	if one == nil {
		panic("Register MQ nil")
	}
	if _, dup := mqs[name]; dup {
		panic("Register MQ duplicate for " + name)
	}
	mqs[name] = one
}

func NewMq(typeName string, url string) (mq, error) {
	if newFun, ok := mqs[typeName]; ok {
		return newFun(url), nil
	}

	return nil, fmt.Errorf("no mq types " + typeName)
}
