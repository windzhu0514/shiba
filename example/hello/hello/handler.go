package hello

import (
	"encoding/json"
	"errors"
	"net/http"
	"runtime/debug"

	"github.com/windzhu0514/shiba/shiba"
	"github.com/windzhu0514/shiba/utils"
)

const (
	ErrCodeOk            = 100
	ErrCodeSign          = 101
	ErrCodeParamMissing  = 102
	ErrCodeParamError    = 103
	ErrCodeInternalError = 104
	ErrCodeNotMatch      = 105
)

type CommonRequest struct {
	Method    string          `json:"reqMethod"` // 操作功能名
	AuthId    string          `json:"authId"`    // 授权 id 服务端预分配
	ReqTime   string          `json:"reqTime"`   // 请求时间 20060102150405
	TraceID   string          `json:"traceId"`   // 签名
	Sign      string          `json:"signature"` // 签名
	ChannelId string          `json:"channelId"`
	Data      json.RawMessage `json:"data,omitempty"`
}

type CommonResponse struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Msg     string      `json:"msg"`
	Data    interface{} `json:"data,omitempty"`
}

type HandlerFunc func(request *CommonRequest) (code int, msg string, data interface{})

func wrapHandle(ctx *shiba.Context, h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			commonRequest  CommonRequest
			commonResponse CommonResponse
		)

		logger := ctx.Logger("wrapHandler")

		defer func() {
			if r := recover(); r != nil {
				buf := debug.Stack()
				logger.Errorf("---------- program crash: %+v ----------", r)
				logger.Errorf("---------- program crash: %s ----------", buf)

				commonResponse.Success = false
				commonResponse.Code = ErrCodeInternalError
				commonResponse.Msg = "内部服务错误"

				w.Write(utils.JsonMarshalByte(commonResponse))

				return
			}
		}()

		jsonStr, err := parseFormJsonStr(r)
		if err != nil {
			commonResponse.Success = false
			commonResponse.Code = ErrCodeParamError
			commonResponse.Msg = "parse http form jsonStr:" + err.Error()

			w.Write(utils.JsonMarshalByte(commonResponse))

			return
		}

		logger.Infof("request jsonStr:%s", jsonStr)

		err = json.Unmarshal([]byte(jsonStr), &commonRequest)
		if err != nil {
			commonResponse.Success = false
			commonResponse.Code = ErrCodeParamError
			commonResponse.Msg = "json解析异常:" + err.Error()

			w.Write(utils.JsonMarshalByte(commonResponse))

			return
		}

		if !ctx.Config().DisableSignatureCheck {
			if !checkSign(commonRequest) {
				commonResponse.Success = false
				commonResponse.Code = ErrCodeSign
				commonResponse.Msg = "sign校验失败"
				w.Write(utils.JsonMarshalByte(commonResponse))

				return
			}
		}

		code, msg, data := h(&commonRequest)
		if code == ErrCodeOk {
			commonResponse.Success = true
		} else {
			commonResponse.Success = false
		}

		commonResponse.Code = code
		commonResponse.Msg = msg
		commonResponse.Data = data

		responseData := utils.JsonMarshalByte(commonResponse)

		logger.Debugf(commonRequest.TraceID+" responseData:%s", responseData)
		w.Write(responseData)
	}
}

func parseFormJsonStr(r *http.Request) (string, error) {
	if err := r.ParseForm(); err != nil {
		return "", err
	}

	jsonStr := r.PostFormValue("jsonStr")
	if jsonStr == "" {
		return "", errors.New("jsonStr is null")
	}

	return jsonStr, nil
}

const ServerAuthKey = "NiD+6Ihdkwie40HxpZmw"

func checkSign(commonRequest CommonRequest) bool {
	sign := utils.MD5(commonRequest.AuthId + commonRequest.Method + commonRequest.ReqTime +
		utils.MD5(commonRequest.Method+commonRequest.ReqTime+ServerAuthKey))

	if commonRequest.Sign == sign {
		return true
	}

	return false

}
