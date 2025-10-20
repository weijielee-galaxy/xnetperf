package server

// Response 统一的API响应结构
type Response struct {
	Code    int         `json:"code"`    // 状态码：0表示成功，非0表示失败
	Message string      `json:"message"` // 提示信息
	Data    interface{} `json:"data"`    // 返回的数据
}

// Success 成功响应
func Success(data interface{}) Response {
	return Response{
		Code:    0,
		Message: "success",
		Data:    data,
	}
}

// SuccessWithMessage 带自定义消息的成功响应
func SuccessWithMessage(message string, data interface{}) Response {
	return Response{
		Code:    0,
		Message: message,
		Data:    data,
	}
}

// Error 错误响应
func Error(code int, message string) Response {
	return Response{
		Code:    code,
		Message: message,
		Data:    nil,
	}
}

// ErrorWithData 带数据的错误响应
func ErrorWithData(code int, message string, data interface{}) Response {
	return Response{
		Code:    code,
		Message: message,
		Data:    data,
	}
}
