package Vortex_Job

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"
)

var (
	//单例对象
	G_apiServer *ApiServer
)


type ApiServer struct {
	httpServer *http.Server
}


type Response struct {
	Code int `json:"code"`
	Status int `json:"status"`
	Data interface{} `json:"data,omitempty"`
	Msg string `json:"msg"`
}

func sendErrResponse(w http.ResponseWriter,code int,msg string)  {
	resp,_ := json.Marshal(&Response{Code:code,Status:1,Msg:msg})
	io.WriteString(w,string(resp))
	return
}

func sendNormalResponse(w http.ResponseWriter,data interface{})  {
	resp,_ := json.Marshal(&Response{Code:http.StatusOK,Status:0,Data:data,Msg:"success"})
	io.WriteString(w,string(resp))
	return
}

//任务的投放
func handlerJobAdd(w http.ResponseWriter,r *http.Request){
	var (
		data []byte
		err error
		)
	if data,err = ioutil.ReadAll(r.Body);err!=nil{
		sendErrResponse(w,431,err.Error())
		return
	}
	if err = G_jobMgr.addJob(data);err!=nil{
		sendErrResponse(w,431,err.Error())
		return
	}

	sendNormalResponse(w,"")

}

//任务的停止
func handlerJobDelete(w http.ResponseWriter,r *http.Request){
	var (
		data []byte
		err error
	)
	if data,err = ioutil.ReadAll(r.Body);err!=nil{
		sendErrResponse(w,431,err.Error())
		return
	}
	if err = G_jobMgr.killJob(data);err!=nil{
		sendErrResponse(w,431,err.Error())
		return
	}

	sendNormalResponse(w,"")

}

//任务的重启
func handlerJobReboot(w http.ResponseWriter,r *http.Request)  {
	var (
		data []byte
		err error
	)
	if data,err = ioutil.ReadAll(r.Body);err!=nil{
		sendErrResponse(w,431,err.Error())
		return
	}
	if err = G_jobMgr.rebootJob(data);err!=nil{
		sendErrResponse(w,431,err.Error())
		return
	}

	sendNormalResponse(w,"")
}

//任务的列出
func handlerJobList(w http.ResponseWriter,r *http.Request){
	var (
		err error
		box []interface{}
	)
	if box,err = G_jobMgr.listJob();err!=nil{
		sendErrResponse(w,431,err.Error())
		return
	}
	if len(box) == 0 {
		sendNormalResponse(w,"")
	}else {
		sendNormalResponse(w,box)
	}

}

//任务的修改
func handlerJobModify(w http.ResponseWriter,r *http.Request)  {
	var (
		data []byte
		err error
	)
	if data,err = ioutil.ReadAll(r.Body);err!=nil{
		sendErrResponse(w,431,err.Error())
		return
	}
	if err = G_jobMgr.modifyJob(data);err!=nil{
		sendErrResponse(w,431,err.Error())
		return
	}

	sendNormalResponse(w,"")
}


//HTTP路由
func RegisterHandlers()(mux *http.ServeMux) {

	mux = http.NewServeMux()

	mux.HandleFunc("/job/add",handlerJobAdd)
	mux.HandleFunc("/job/stop",handlerJobDelete)
	mux.HandleFunc("/job/reboot",handlerJobReboot)
	mux.HandleFunc("/job/list",handlerJobList)
	mux.HandleFunc("/job/modify",handlerJobModify)

	return
}

func InitApiServer()(err error){
	var(
		mux *http.ServeMux
		listener net.Listener
		httpServer *http.Server
	)

	ApiPort := G_config.ApiPort
	ApiReadTimeout:= G_config.ApiReadTimeout
	ApiWriteTimeout:= G_config.ApiWriteTimeout

	//路由配置
	mux = RegisterHandlers()

	//启动TCP监听
	if listener,err = net.Listen("tcp",":"+strconv.Itoa(ApiPort));err!=nil{
		return
	}

	//创建一个http服务
	httpServer = &http.Server{
		ReadTimeout:time.Duration(ApiReadTimeout)*time.Millisecond,
		WriteTimeout:time.Duration(ApiWriteTimeout)*time.Millisecond,
		Handler:mux,
	}

	//赋值单例
	G_apiServer = &ApiServer{
		httpServer:httpServer,
	}

	//启动服务端
	fmt.Println("HTTP服务开启在端口:",ApiPort)
	go httpServer.Serve(listener)

	return
}
