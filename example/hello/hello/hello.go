package hello

import (
	"encoding/json"
	"fmt"

	"github.com/windzhu0514/shiba/shiba"
)

type config struct {
	Str string `yaml:"str"`
}

type Hello struct {
	Config   config `yaml:"hello"`
	FlagName string
}

func (h *Hello) Name() string {
	return "hello"
}

func (h *Hello) Init(ctx *shiba.Context) error {
	ctx.Router().Handle("/hello", wrapHandle(ctx, h.helloHandler))
	ctx.FlagSet().StringVar(&h.FlagName, "name", "", "hello to who")
	return nil
}

func (h *Hello) Start(ctx *shiba.Context) error {
	fmt.Printf("hello %s service is start:%s\n", h.FlagName, h.Config.Str)
	//db, err := server.DBSlave("localhost")
	//if err != nil {
	//	return err
	//}
	//var count sql.NullInt32
	//err = db.QueryRowx("select count(*) from mobile_deviceinfo_template").Scan(&count)
	//if err != nil {
	//	return err
	//}
	//
	//fmt.Println(count)

	return nil
}

func (h *Hello) Stop(ctx *shiba.Context) error {
	fmt.Println("hello service is stop:" + h.Config.Str)
	ctx.Logger("hello").Error("hahhahah hello")
	return nil
}

type HelloReqData struct {
}

type HelloRespData struct {
}

func (h *Hello) helloHandler(request *CommonRequest) (code int, msg string, data interface{}) {
	var requestData HelloReqData
	err := json.Unmarshal(request.Data, &requestData)
	if err != nil {
		return ErrCodeParamError, "json解析异常:" + err.Error(), nil
	}

	var respData HelloRespData

	return ErrCodeOk, "", respData
}
