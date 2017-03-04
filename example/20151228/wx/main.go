/**
 * 微信API示例 by woylin 2015/12/28
 */
package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	_ "github.com/alexbrainman/odbc"
	"github.com/esap/wechat" // 微信SDK包
	"github.com/esap/wechat/util"
	"github.com/jmoiron/sqlx"
)

const (
	FTP_PATH   = `E:\esdisk\hr\` //网盘路径，ES需启用网盘功能，在系统网盘根目录下建立了hr目录
	PIC_PREFIX = "P00"           //照片前缀，默认是"P00"
	PIC_SUFFIX = ".jpg"          //照片后缀，默认".jpg"
)

var (
	Db     *sqlx.DB
	err    error
	EmpMap = make(map[string]string)
)

func main() {
	dsn := fmt.Sprintf("driver={SQL Server};SERVER=%s;UID=%s;PWD=%s;DATABASE=%s",
		"192.168.99.20", "sa", "123", "esapp1")
	if Db, err = sqlx.Connect("odbc", dsn); err != nil {
		log.Fatal(err)
	}
	wechat.Set("esap", "wxf0ff4b47419f81b1", "d4624c36b6795d1d99dcf0547af5443d")
	http.HandleFunc("/", WxHandler)
	log.Fatal(http.ListenAndServe(":9090", nil))
}

// HTTP默认处理函数
func WxHandler(w http.ResponseWriter, r *http.Request) {
	ctx := wechat.VerifyURL(w, r)
	if ctx == nil {
		return
	}
	log.Println("user-msg:", ctx.Msg)
	switch ctx.Msg.MsgType {
	case "text":
		GoText(ctx)
	case "image":
		GoImage(ctx)
	case "event":
		if ctx.Msg.Event == "subscribe" {
			ctx.NewText("欢迎关注").Reply()
		}
	}
}

// 图片消息处理
func GoImage(w *wechat.Context) {
	EmpMap[w.Msg.FromUserName] = w.Msg.PicUrl
	w.NewText("请输入员工信息(格式 姓名,工号)：").Reply()
}

// 文本消息处理
func GoText(w *wechat.Context) {
	// 1.检查是否已上传图片，上传过则提示输入“姓名，工号”
	if picUrl, ok := EmpMap[w.Msg.FromUserName]; ok {
		// 存在未完流程则进入照片处理流程
		w.NewText(handPic(w, picUrl)).Reply()
		return
	}
	// 2.未上传图片其他消息的处理，仅示例
	switch w.Msg.Content {
	case "蛋蛋":
		// 创建三个文章
		art1 := wechat.NewArticle("ESAP第十五弹 玩转微信企业号对接ES",
			"来自村长的ESAP系统最新技术分享。",
			"https://iesap.net/img/esap15-1.jpg",
			"https://iesap.net/2016/01/28/esap15/")
		art2 := wechat.NewArticle("ESAP第十四弹 手把手教你玩转ES微信开发",
			"来自村长的ESAP系统最新技术分享。",
			"https://iesap.net/img/esap14-1.jpg",
			"https://iesap.net/2015/12/28/esap14/")
		art3 := wechat.NewArticle("我与ES不吐不快的槽",
			"来自村长的工作日志。",
			"https://iesap.net/img/esdiediedie-2.jpg",
			"https://iesap.net/2014/10/31/EsDiediedie/")
		// 创建文章回复
		w.NewNews(art1, art2, art3).Reply()
	default:
		// 创建默认文本回复
		w.NewText("hi，欢迎加入ESAP部落/::D").Reply()
	}
}

// 照片处理函数
func handPic(w *wechat.Context, picUrl string) string {
	// 解析姓名工号
	empInfo := strings.Split(strings.Replace(w.Msg.Content, "，", ",", 1), ",")
	if len(empInfo) != 2 {
		return "你填入的信息格式不正确，请重新输入"
	}
	// 通过employee表查找rcid,未找到则提示用户。
	var rcid string
	if err := Db.Get(&rcid, "select Excelserverrcid from employee where name=? and eid=?",
		empInfo[0], empInfo[1]); err != nil {
		return "查询员工错误:" + err.Error()
	}
	// 设置照片名为：前缀 + 工号
	picName := PIC_PREFIX + empInfo[1]
	// 删除原有数据库照片路径记录
	if _, err := Db.Exec("delete from es_casepic where rcid=?", rcid); err != nil {
		return "图片删除失败:" + err.Error()
	}
	// 向数据库插入新照片路径记录,这里的ed\esdisk,wx\hr等信息要改成自己的ES网盘配置目录
	if _, err := Db.Exec("insert es_casepic(emp.Rcid,picNo,fileType,rtfid,sh,r,c,saveinto,nfsfolderid,nfsfolder,relafolder,phyfileName) values(?,?,?,?,?,?,?,?,?,?,?,?)",
		rcid, picName, ".jpg", 60, 1, 3, 4, 1, 1, `me\sys`, `hr\`, picName+".jpg"); err != nil {
		return "图片上传失败:" + err.Error()
	}
	// 上传图片到网盘，销毁员工数组中的信息，回复处理成功信息
	fn := FTP_PATH + PIC_PREFIX + empInfo[1] + PIC_SUFFIX //设置照片存放路径及文件名
	if err := util.GetFile(fn, picUrl); err != nil {
		return "图片下载失败:" + err.Error()
	}
	delete(EmpMap, w.Msg.FromUserName)
	return "员工照片已成功处理"
}
