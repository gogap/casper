package casper

import (
	"github.com/gogap/errors"

	"github.com/gogap/casper/errorcode"
)

var mqs map[string]mqType = make(map[string]mqType)

// 消息接口
type MessageQueue interface {
	Ready() error                   // 初始化
	RecvMessage() ([]byte, error)   // 读一条消息
	SendToNext([]byte) (int, error) // 发送一条消息到下一节点
}

type mqType func(string) MessageQueue

func registerMq(name string, one mqType) {
	if one == nil {
		panic("Register MQ nil")
	}
	if _, dup := mqs[name]; dup {
		panic("Register MQ duplicate for " + name)
	}
	mqs[name] = one
}

func NewMQ(compMeta *ComponentMetadata) (mq MessageQueue, err error) {

	if compMeta == nil {
		return nil, errorcode.ERR_COMPONENT_METADATA_IS_NIL.New()
	}

	if compMeta.MQType == "" {
		return nil, errorcode.ERR_COMPONENT_MQTYPE_IS_EMPTY.New(
			errors.Params{"name": compMeta.Name})
	}

	if compMeta.In == "" {
		return nil, errorcode.ERR_COMPONENT_IN_IS_EMPTY.New(
			errors.Params{"name": compMeta.Name})
	}

	if newFun, ok := mqs[compMeta.MQType]; ok {
		return newFun(compMeta.In), nil
	}

	err = errorcode.ERR_COULD_NOT_NEW_MSG_QUEUE.New(
		errors.Params{
			"name":   compMeta.Name,
			"mqType": compMeta.MQType})

	return nil, err
}
