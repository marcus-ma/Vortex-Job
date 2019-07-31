package main

import (
	Vortex_Job "Vortex-Job"
	"flag"
	"fmt"
	"runtime"
	"time"
)

var (
	//配置文件路径
	confFile string
)

//Vortex-Job
//Vortex-Job is a scheduled job tool, based on Etcd
func initArgs()  {
	// master -config /master.json
	flag.StringVar(&confFile,"config","./config.json","请指定config.json")
	flag.Parse()
}
//初始化线程数量
func initEnv()  {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	var (err error)

	//初始化命令行参数
	initArgs()

	//初始化线程
	initEnv()

	// 加载配置(从命令行参数获取)
	if err = Vortex_Job.InitConfig(confFile);err!=nil{
		goto ERR
	}

	// 初始化任务调度器
	if err = Vortex_Job.InitScheduler();err!=nil{
		goto ERR
	}

	// 初始化任务管理器
	if err = Vortex_Job.InitJobMgr();err!=nil{
		goto ERR
	}

	// 初始化任务执行器
	if err = Vortex_Job.InitExecutor();err!=nil{
		goto ERR
	}

	// 初始化API服务器
	if err = Vortex_Job.InitApiServer();err!=nil{
		goto ERR
	}

	//正常退出
	for  {
		time.Sleep(time.Second*1)
	}


ERR:
	fmt.Println(err)

}
