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
)
