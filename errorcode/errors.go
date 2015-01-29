package errorcode

import (
	"github.com/gogap/errors"
)

var (
	ERR_API_NOT_FOUND = errors.T(404, "api of {{.apiName}} not found.")
	ERR_BAD_REQUEST   = errors.T(400, "bad request, path: {{.path}}, raw error is: {{.err}}")

	ERR_JSON_MARSHAL_ERROR          = errors.T(1000, "marshal json error, raw error is: {{.err}}")
	ERR_REQUEST_SHOULD_BE_JSON      = errors.T(1001, "request data should be json format")
	ERR_COULD_NOT_NEW_COMPONENT_MSG = errors.T(1002, "could not new a component message, raw error is: {{.err}}")
	ERR_SEND_COMPONENT_MSG_ERROR    = errors.T(1003, "send component message error, id: {{.id}}, raw error is: {{.err}}")

	ERR_OPENFILE_ERROR           = errors.T(1004, "open file error, raw error is: {{.fileName}}, raw error is: {{.err}}")
	ERR_JSON_UNMARSHAL_ERROR     = errors.T(1005, "unMarshal json error, raw error is: {{.err}}")
	ERR_COMPONENT_NOT_EXIST      = errors.T(1006, "component not exist, name: {{.name}}")
	ERR_COMPONENT_HANDLER_IS_NIL = errors.T(1007, "component handler is nil")

	ERR_COMPONENT_METADATA_IS_NIL = errors.T(1008, "component metadata is nil")
	ERR_COMPONENT_MQTYPE_IS_EMPTY = errors.T(1009, "component mqtype is empty, name: {{.name}}")
	ERR_COMPONENT_IN_IS_EMPTY     = errors.T(1010, "component in is empty, name: {{.name}}")
	ERR_COULD_NOT_NEW_MSG_QUEUE   = errors.T(1011, "create new msg queue error, name: {{.name}}, MQType: {{.mqType}}")

	ERR_ZMQ_URL_IS_EMPTY             = errors.T(1012, "zmq's url is empty")
	ERR_NEW_ZMQ_FAILED               = errors.T(1013, "could not new zmq socket, url: {{.url}}, type: {{.type}}, raw error is: {{.err}}")
	ERR_ZMQ_COULD_NOT_BIND_URL       = errors.T(1014, "could not bind zmq socket to {{.url}}, type: {{.type}}, raw error is: {{.err}}")
	ERR_ZMQ_COULD_NOT_CONNECT_TO_URL = errors.T(1015, "could not connect to zmq socket to {{.url}}, type: {{.type}}, raw error is: {{.err}}")
	ERR_ZMQ_RECV_MSG_FAILED          = errors.T(1016, "recv zmq message failed, url: {{.url}}, raw error is: {{.err}}")
	ERR_ZMQ_RECV_MSG_INVALID         = errors.T(1017, "recv zmq message is invalid, url: {{.url}}")

	ERR_MESSENGER_REQ_ID_NOT_EXIST     = errors.T(1018, "messagener request id not exist, request id: {{.id}}, msg: {{.msg}}")
	ERR_COULD_NOT_PARSE_COMPONENT_MSG  = errors.T(1019, "could not parse component message, IN: {{.in}}, MQType: {{.mqType}}, msg: {{.msg}}")
	ERR_HANDLER_RETURN_ERROR           = errors.T(1020, "handler return error, component name: {{.name}}")
	ERR_COMPONENT_MSG_SERIALIZE_FAILED = errors.T(1021, "component serialize failed, IN: {{.in}}, MQType: {{.mqType}}, raw error: {{.err}}")

	ERR_GRAPH_NOT_EXIST = errors.T(1022, "graph {{.name}} not exist")
)
