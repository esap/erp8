/**
 * 企业号API实例-应用分支实现
 * @woylin, since 2016-1-6
 */
package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/esap/sqlsrv"
	"github.com/esap/wechat"
)

/**
 * 应用模板 - 库存查询
 * 用户点击进入时会提示近期到料情况
 * 用户输入关键字，可查询物料描述或批号包含关键的的库存信息
 */
type AgentKC struct {
	WxAgent
}

func (w *AgentKC) Gtext() {
	w.ctx.NewText("正在查询库存情况...").Reply()
	go kc(w.ctx.Msg.FromUserName, w.ctx.Msg.AgentID, w.ctx.Msg.Content)
}

func (w *AgentKC) Gevent() {
	if w.ctx.Msg.Event == "enter_agent" {
		w.ctx.NewText("近期到料情况：")
		go jqdl(w.ctx.Msg.FromUserName, w.ctx.Msg.AgentID)
	}
}

// 定义库存查询字段，即：描述，数量
type lbq struct {
	Mdesc string
	Qty   float32
}

// 定义库存查询输出格式
func (c lbq) String() string {
	return fmt.Sprintf("%v\n= %v\n", c.Mdesc, c.Qty)
}

// 库存查询
func kc(user string, id int, mDesc string) {
	sql := fmt.Sprintf("select 名 + '/' + 批,数 from vlbq where (charindex('%s',名)>0 or charindex('%s',批)>0)", mDesc, mDesc)
	queryAndSendArr(user, id, sql, &lbq{})
}

// 定义备件到料情况字段
type dl struct {
	GrDate time.Time
	Mdesc  string
	Rem    string
	Qty    float32
	Unit   string
}

// 到料输出格式
func (c dl) String() string {
	return fmt.Sprintf("%v-%v%v\n到料 = %v %v\n", c.GrDate.Format("1/2"), c.Mdesc, c.Rem, c.Qty, c.Unit)
}

// 近期到料
func jqdl(user string, id int) {
	queryAndSendArr(user, id, "select 日期,名,批,sum(数),单位 from io,s where io.ExcelServerRCID=s.ExcelServerRCID group by 日期,名,批,单位", &dl{})
}

/**
 * 应用模板 - ESAP示例
 * 示例演示了各种信息的回复方法
 */
type AgentESAP struct {
	WxAgent
}

func (w *AgentESAP) Gtext() {
	w.ctx.ReplySuccess()
	wechat.SendText(w.ctx.Msg.FromUserName, w.ctx.Msg.AgentID, w.ctx.Msg.Content)
}
func (w *AgentESAP) Gimage() {
	w.ctx.NewImage(w.ctx.Msg.MediaId).Reply() //图片消息
}
func (w *AgentESAP) Gvoice() {
	w.ctx.NewVoice(w.ctx.Msg.MediaId).Reply() // 语音消息
}
func (w *AgentESAP) Gshortvideo() {
	w.ctx.NewVideo(w.ctx.Msg.MediaId, "看一看", "瞧一瞧").Reply() // 短视频消息
}
func (w *AgentESAP) Gvideo() {
	w.ctx.NewVideo(w.ctx.Msg.MediaId, "再看一看", "再瞧一瞧").Reply() // 视频消息
}
func (w *AgentESAP) Glocation() {
	w.ctx.NewText("本次签到地点：" + w.ctx.Msg.Label) // 位置签到消息
}
func (w *AgentESAP) Glink() {
	w.ctx.NewText(w.ctx.Msg.Title + w.ctx.Msg.Url) // 链接消息
}
func (w *AgentESAP) Gevent() {

	art := wechat.NewArticle("ESAP第十五弹 玩转微信企业号对接ES",
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
	w.ctx.NewNews(art, art2, art3).Reply()

}

/**
 * 应用模板 - 照片采集
 * 用户先上传或拍摄照片，然后按特定格式填入姓名，工号
 * 经过数据库匹配后，完成图片采集和更新
 * 数据库插入图片路径时需大量字段匹配，例如图片字段的sheet,row,column...都需要一一匹配
 */
type AgentPIC struct {
	WxAgent
}

func (w *AgentPIC) Gtext() {
	// 匹配姓名工号并存入ESAP
	w.ctx.NewText("正在处理...")
	go zpcl(w.ctx.Msg)
}

func (w *AgentPIC) Gimage() {
	// 接收图片，提示录入
	empMap[w.ctx.Msg.FromUserName] = &employee{"", "", w.ctx.Msg.PicUrl, nil, 0}
	w.ctx.NewText("请输入姓名，工号\n例如：“张三，120”")
}

func (w *AgentPIC) Gevent() {
	if w.ctx.Msg.Event == "enter_agent" {
		w.ctx.NewText("请拍摄或选择相册照片发送后，填写姓名，工号。")
	}
}

const (
	FTPPATH   = `R:\hr\emp\` // 相对本程序服务器的照片存放路径，ES需启用网盘功能
	ESDISK    = `ed\wx`      // ES网盘根目录，管理控制台
	SUBPATH   = `hr\emp\`    // 照片存放子目录
	PICPREFIX = "P00"        // 照片前缀，默认是"P00"
	PICSUFFIX = ".jpg"       // 照片后缀，默认".jpg"
	// 下面这些字段可以通过正常插入照片后 select top 2 * from es_casepic order by rcid desc 仿照填入^_^
	RtfId       = 1 // 图片字段id
	sh          = 1 // 图片字段sheet
	row         = 1 // 图片字段row
	column      = 1 // 图片字段column
	SaveInto    = 1 // 网盘号
	NFSFolderId = 1 // 根目录号
)

// 定义员工
type employee struct {
	Name     string
	Eid      string
	Photo    string
	Rcid     interface{}
	errCount int
}

// 员工照片上传，可为网盘
func (e *employee) download(url string) {
	imgResp, _ := http.Get(url) // 从微信服务器获取照片
	defer imgResp.Body.Close()
	m, _, _ := image.Decode(imgResp.Body)         // 解析照片信息
	fn := FTPPATH + PICPREFIX + e.Eid + PICSUFFIX // 设置照片存放路径及文件名
	fh, _ := os.Create(fn)                        // 重建照片
	defer fh.Close()
	jpeg.Encode(fh, m, nil) //以JPG格式保存照片
}

// 定义员工表、存放照片链接信息
var empMap = make(map[string]*employee)

// 照片处理
func zpcl(w *wechat.WxMsg) {
	// 检验是否已上传图片，上传过则提示输入“姓名，工号”
	var bd string
	if v, ok := empMap[w.FromUserName]; ok {
		// 解析姓名工号
		empInfo := strings.Split(w.Content, "，")
		if len(empInfo) == 2 {
			v.Name, v.Eid = empInfo[0], empInfo[1]
		} else {
			// 重复出错4次后重置，取消流程
			v.errCount++
			if v.errCount > 3 {
				bd = "出错已超限，上传流程已取消"
				delete(empMap, w.FromUserName)
			} else {
				bd = fmt.Sprintf("你填入的信息格式不正确，请重新输入。(%d)", v.errCount)
			}
			wechat.SendText(w.FromUserName, w.AgentID, bd)
			return
		}
		// 通过员工信息表查找rcid,未找到则提示用户。
		v.Rcid = sqlsrv.Fetch("select Excelserverrcid from 员工信息表 where 姓名=? and 工号=?", v.Name, v.Eid)
		if (v.Rcid) == nil {
			bd = "未找到该用户，请重新输入或重新上传图片。"
		} else {
			// 设置照片名为：前缀 + 工号
			picName := PICPREFIX + v.Eid
			// 删除原有数据库照片路径记录，r,c是照片的Excel行列号
			sqlsrv.Exec("delete from es_casepic where rcid=? and r=? and c=?", v.Rcid, row, column)
			// 向数据库插入新照片路径记录
			err := sqlsrv.Exec("insert es_casepic(rcid,picNo,fileType,rtfid,sh,r,c,saveinto,nfsfolderid,nfsfolder,relafolder,phyfileName) values(?,?,?,?,?,?,?,?,?,?,?,?)",
				v.Rcid, picName, ".jpg", RtfId, sh, row, column, SaveInto, NFSFolderId, ESDISK, SUBPATH, picName+".jpg")
			if err != nil {
				bd = "图片上传失败"
			} else {
				// 上传图片到网盘，销毁员工数组中的信息，回复处理成功信息
				v.download(v.Photo)
				delete(empMap, w.FromUserName)
				bd = "员工照片已成功处理"
			}
		}
		wechat.SendText(w.FromUserName, w.AgentID, bd)
	}
}

/**
 * 应用模板 - 待办事宜
 * 基于通用审核 https://iesap.net/2015/07/28/esap11/
 * 用户点击下一条按钮获取一条通用待办信息，点击通过或不通过后生成审核记录更新通用待办视图
 */
type AgentDB struct {
	WxAgent
}

// 销售订单
type sxDd struct {
	No     string
	Cdate  time.Time
	Cre    string
	Seller string
	Mdesc  string
	Qty    float32
	Mprice float32
	Rem    string
}

var mapSxDd = make(map[string]*sxDd)

func (c sxDd) String() string {
	return fmt.Sprintf("订单号：%v\n下单日期：%v\n创建人：%v\n业务员：%v\n产品描述：%v\n订单数：%v\n单价：%v\n备注：%v\n",
		c.No, c.Cdate.Format("2006-1-2"), c.Cre, c.Seller, c.Mdesc, c.Qty, c.Mprice, c.Rem)
}

func (w *AgentDB) Gevent() {
	switch w.ctx.Msg.Event {
	case "enter_agent":
		w.ctx.NewText("温馨提示：点击分类即可开始逐条办理。")
	case "click":
		switch w.ctx.Msg.EventKey {
		case "next":
			arr := sqlsrv.FetchOnePtr("select dNo,cdate,c,seller,mdesc,qty,mprice,rem from o2a where sgid=?", &sxDd{}, 8001)
			v := (*arr).(sxDd)
			mapSxDd[w.ctx.Msg.FromUserName] = &v
			log.Printf("---arr:%v", *arr)
			w.ctx.NewText(fmt.Sprintf("%v", *arr))
		case "yes":
			if k, ok := mapSxDd[w.ctx.Msg.FromUserName]; ok {
				log.Println("--rec found:", k)
				err := sqlsrv.Exec("insert into oda(oNo,V,T,rem,cre,oDate,s) values(?,?,?,?,?,?,?)",
					k.No, "Y", 30, "", w.ctx.Msg.FromUserName, time.Now().Format("2006-1-2 15:04:05"), 0)
				if err != nil {
					w.ctx.NewText("审核失败，数据处理异常。")
				}
				w.ctx.NewText("审核成功。")
				delete(mapSxDd, w.ctx.Msg.FromUserName)
			}
		case "no":
			if k, ok := mapSxDd[w.ctx.Msg.FromUserName]; ok {
				log.Println("--rec found:", k)
				err := sqlsrv.Exec("insert into oda(oNo,V,T,rem,cre,oDate,s) values(?,?,?,?,?,?,?)",
					k.No, "N", 30, "", w.ctx.Msg.FromUserName, time.Now().Format("2006-1-2 15:04:05"), 0)
				if err != nil {
					w.ctx.NewText("审核失败，数据处理异常。")
				}
				w.ctx.NewText("审核成功。").Reply()
				delete(mapSxDd, w.ctx.Msg.FromUserName)
			}
		}
	}
}

func queryMaxDate(user string, id int, table string) {
	date := sqlsrv.Fetch(fmt.Sprintf("select max(cdate) from %s", table))
	if v, ok := (*date).(time.Time); ok {
		wechat.SendText(user, id, fmt.Sprintf("最新记录日期: %s", v.Format("2006-01-02")))
	}
}

// 通用方法，逐条回复sql查询到的内容
func queryAndSend(user string, id int, sql string, struc interface{}) {
	time.Sleep(time.Second)
	arr := sqlsrv.FetchAllRowsPtr(sql, struc)
	if len(*arr) == 0 {
		wechat.SendText(user, id, "未找到项目...")
	}
	log.Println("--arr:", *arr)
	for k, v := range *arr {
		s := fmt.Sprintf("%v：%v", k+1, v)
		if len(*arr) == 1 {
			s = fmt.Sprintf("%v", v)
		}
		wechat.SendText(user, id, s)
	}

}

//通用方法，合并回复sql查询到的内容（更常用）
func queryAndSendArr(user string, id int, sql string, struc interface{}) {
	time.Sleep(time.Second)
	arr := sqlsrv.FetchAllRowsPtr(sql, struc)
	if len(*arr) == 0 {
		wechat.SendText(user, id, "未找到项目...")
	}
	log.Println("--arr:\n", *arr)
	s := strings.TrimSuffix(strings.TrimPrefix(fmt.Sprintf("%v", *arr), "["), "]")
	wechat.SendText(user, id, s)
}
