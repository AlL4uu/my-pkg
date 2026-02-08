package ginx

type Result struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func Success(data interface{}) Result {
	return Result{Code: 0, Msg: "success", Data: data}
}

func Error(code int, msg string) Result {
	return Result{Code: code, Msg: msg}
}
