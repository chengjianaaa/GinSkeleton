package websocket

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"goskeleton/app/global/variable"
	"goskeleton/app/utils/websocket/core"
	"time"
)

/**
websocket模块相关事件执行顺序：
1.onOpen
2.OnMessage
3.OnError
4.OnClose
*/

type Ws struct {
	WsClient *core.Client
}

// onOpen 基本不需要做什么
func (w *Ws) OnOpen(context *gin.Context) (*Ws, bool) {
	if client, ok := (&core.Client{}).OnOpen(context); ok {
		w.WsClient = client
		go w.WsClient.Heartbeat(w.OnClose) // 一旦握手+协议升级成功，就为每一个连接开启一个自动化的隐式心跳检测包
		return w, true
	} else {
		return nil, false
	}
}

// OnMessage 处理业务消息
func (w *Ws) OnMessage(context *gin.Context) {
	go w.WsClient.ReadPump(func(message_type int, received_data []byte) {
		//参数说明
		//message_type 消息类型，1=文本
		//received_data 服务器接收到客户端（例如js客户端）发来的的数据，[]byte 格式

		v_temp_msg := "服务器已经收到了你的消息==>" + string(received_data)
		w.WsClient.Conn.WriteMessage(message_type, []byte(v_temp_msg)) // 回复客户端已经收到消息

	}, w.OnError, w.OnClose)
}

// OnError 客户端与服务端在消息交互过程中发生错误回调函数
func (w *Ws) OnError(err error) {
	variable.ZapLog.Error("远端掉线、卡死、刷新浏览器等会触发该错误:", zap.Error(err))
	//fmt.Printf("远端掉线、卡死、刷新浏览器等会触发该错误: %v\n", err.Error())
}

// OnClose 客户端关闭回调，发生onError回调以后会继续回调该函数
func (w *Ws) OnClose() {

	w.WsClient.Hub.UnRegister <- w.WsClient // 向hub管道投递一条注销消息，有hub中心负责关闭连接、删除在线数据
}

//获取在线的全部客户端
func (w *Ws) GetOnlineClients() {

	fmt.Printf("在线客户端数量：%d\n", len(w.WsClient.Hub.Clients))
}

// 向全部在线客户端广播消息
func (w *Ws) BroadcastMsg(send_msg string) {

	for online_client, _ := range w.WsClient.Hub.Clients {

		online_client.Conn.SetWriteDeadline(time.Now().Add(w.WsClient.WriteDeadline * time.Second)) // 每次向客户端写入消息命令（WriteMessage）之前必须设置超时时间
		online_client.Conn.WriteMessage(websocket.TextMessage, []byte(send_msg))                    //获取每一个在线的客户端，向远端发送消息
	}
}