package api

const (
	// 统一错误码定义。
	ErrCodeSuccess      = 0
	ErrCodeBadRequest   = 1001
	ErrCodeInternal     = 1002
	ErrCodeNotFound     = 1003
	ErrCodeUnauthorized = 1004
	ErrCodeConflict     = 1005
)

type Response struct {
	// 标准 HTTP 响应结构。
	Errno  int         `json:"errno"`
	Errmsg string      `json:"errmsg"`
	Data   interface{} `json:"data,omitempty"`
}

func SuccessResponse(data interface{}) Response {
	// 成功响应。
	return Response{
		Errno:  ErrCodeSuccess,
		Errmsg: "success",
		Data:   data,
	}
}

func ErrorResponse(errno int, errmsg string) Response {
	// 通用错误响应。
	return Response{
		Errno:  errno,
		Errmsg: errmsg,
		Data:   nil,
	}
}

func BadRequestResponse(errmsg string) Response {
	// 400 错误响应。
	return Response{
		Errno:  ErrCodeBadRequest,
		Errmsg: errmsg,
		Data:   nil,
	}
}

func InternalErrorResponse(errmsg string) Response {
	// 500 错误响应。
	return Response{
		Errno:  ErrCodeInternal,
		Errmsg: errmsg,
		Data:   nil,
	}
}
