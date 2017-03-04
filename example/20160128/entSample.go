/**
 * 企业号API实例.
 * @woylin, since 2016-1-6
 */
package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/esap/sqlsrv"
	"github.com/esap/wechat"
)

var (
	token          = "esap"
	corpId         = "wxcde58ce7814e8414"
	encodingAesKey = "ehHEULdSLc67a5Ol6wQZ7WPWev1mqug72nvLqIMOYyv"
	secret         = "qofNzZmkmw3pWKv1z8Hh6Gwt97XqpD4JzH9KIl9EsG1qiPzgKoxHAjc_ZGQDaOWv"
	agentMap       = make(map[int]WxAgenter)
)

func main() {
	// 注册应用分支
	//	agentMap[1] = &AgentPIC{} // 采集照片
	//	agentMap[2] = &AgentDB{} // 待办
	agentMap[15] = &AgentESAP{} //ESAP
	agentMap[12] = &AgentKC{}   // 备件

	// 设置wechat参数
	wechat.Debug = true
	wechat.SetEnt(token, corpId, secret, encodingAesKey)

	http.HandleFunc("/wx", wxHander)
	log.Fatal(http.ListenAndServe(":9090", nil))
}

// 微信“应用接口”，实现这些接口可被API主控引导调用
type WxAgenter interface {
	Gtext()
	Gimage()
	Gvoice()
	Gshortvideo()
	Gvideo()
	Glocation()
	Glink()
	Gevent()
	SetMsg(*wechat.Context)
}

//实现微信“应用接口”的父应用，定义应用时应继承
type WxAgent struct {
	ctx *wechat.Context
}

//实现接口的函数， 默认为空方法
func (w *WxAgent) Gtext()                   {}
func (w *WxAgent) Gimage()                  {}
func (w *WxAgent) Gvoice()                  {}
func (w *WxAgent) Gshortvideo()             {}
func (w *WxAgent) Gvideo()                  {}
func (w *WxAgent) Glocation()               {}
func (w *WxAgent) Glink()                   {}
func (w *WxAgent) Gevent()                  {}
func (w *WxAgent) SetMsg(b *wechat.Context) { w.ctx = b }

// API主控
func wxHander(w http.ResponseWriter, r *http.Request) {
	ctx := wechat.VerifyURL(w, r)
	if ctx.Msg == nil {
		log.Println("invalid ctx...")
		return
	}
	fmt.Printf("%+v", ctx.Msg)
	// 查找已注册的应用，未找到则提示该应用未实现
	agent, ok := agentMap[ctx.Msg.AgentID]
	if !ok {
		log.Printf("--未找到ID[%v]的应用分支!\n", ctx.Msg.AgentID)
		return
	}
	// 传递微信请求到应用
	agent.SetMsg(ctx)
	// 根据微信请求类型（MsgType），调用应用接口进行处理
	switch ctx.Msg.MsgType {
	case "text":
		agent.Gtext()
	case "image":
		agent.Gimage()
	case "voice":
		agent.Gvoice()
	case "shortvideo":
		agent.Gshortvideo()
	case "video":
		agent.Gvideo()
	case "location":
		agent.Glocation()
	case "link":
		agent.Glink()
	case "event":
		agent.Gevent()
	}
}

// 微信提醒
type wxtx struct {
	ToUser  string
	ToAgent int
	Context string
	Id      int
}

// 循环扫描微信提醒
func checkWxtx() {
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			log.Println("Scanning msg to send")
			arr := sqlsrv.FetchAllRowsPtr("select touser,toagent,context,id from wxtx where isnull(flag,0)=0", &wxtx{})
			if len(*arr) == 0 {
				continue
			}
			for _, v := range *arr {
				if v1, ok := v.(wxtx); ok {
					s := fmt.Sprintf("【新待办通知】\n描述：%v\n", v1.Context)
					log.Printf("--msg to send:%v", s)
					wechat.SendText(v1.ToUser, v1.ToAgent, s)
					sqlsrv.Exec("update wxtx set flag=1 where id=?", v1.Id)
				}
			}
		}
	}()
}
