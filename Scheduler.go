package Vortex_Job

import (
	"fmt"
	"time"
)

type Scheduler struct {
	jobEventChan chan *JobEvent	// 任务事件队列
	jobPlanTable map[string]*JobSchedulePlan // 任务调度计划表
	jobExecutingTable map[string]*JobExecuteInfo // 任务执行表
	jobResultChan chan *JobExecuteResult	// 任务结果队列
}

var (G_scheduler *Scheduler)


//调度协程
func (scheduler *Scheduler)schedulerLoop(){
	var (
		jobEvent *JobEvent
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
		jobResult *JobExecuteResult
	)
	//初始化时间间隔(1s)
	scheduleAfter = scheduler.TryScheduler()

	//调度的延迟定时器
	scheduleTimer = time.NewTimer(scheduleAfter)

	//定时任务
	for {
		select {
		//监听任务变化事件
		case jobEvent = <- scheduler.jobEventChan :
			//对内存中维护的任务列表做增删改查
			scheduler.handleJobEvent(jobEvent)
		//最近的任务过期了
		case <-scheduleTimer.C:
			//监听任务执行结果
		case jobResult = <- scheduler.jobResultChan:
			scheduler.handleJobResult(jobResult)
		}
		//调度一次任务
		scheduleAfter = scheduler.TryScheduler()
		//重置调度间隔
		scheduleTimer.Reset(scheduleAfter)
	}
}
//重新计算任务调度状态
func (scheduler *Scheduler)TryScheduler() (scheduleAfter time.Duration) {
	var (
		jobPlan *JobSchedulePlan
		now time.Time
		nearTime *time.Time
	)
	//若当前任务列表为空，睡眠1s
	if len(scheduler.jobPlanTable)==0{
		scheduleAfter = 1 * time.Second
		return
	}

	//当前时间
	now = time.Now()
	//1.便遍历所有任务
	for _,jobPlan = range scheduler.jobPlanTable{
		//2.过期任务立即执行
		if jobPlan.NextTime.Before(now)||jobPlan.NextTime.Equal(now){
			scheduler.TryStartJob(jobPlan)
			//更新下次执行的时间
			jobPlan.NextTime = jobPlan.Expr.Next(now)
		}
		//3.统计最近的要过期的任务的时间(N秒后过期 == scheduleAfter)
		//统计最近一个将要过期的任务时间
		if nearTime == nil||jobPlan.NextTime.Before(*nearTime){
			nearTime = &jobPlan.NextTime
		}
	}
	//下次调度间隔(最近要执行的任务调度时间 - 当前时间)
	scheduleAfter = (*nearTime).Sub(now)
	return
}
//尝试执行任务
func(scheduler *Scheduler)TryStartJob(jobPlan *JobSchedulePlan)   {
	//调度与执行为2件事
	//执行的任务可能运行很久，1分钟会调度60次，但只能执行1次，防止并发！
	var (
		jobExecuteInfo *JobExecuteInfo
		jobExecuting bool
	)

	//判断任务是否在执行，否则跳过本次调度
	if jobExecuteInfo,jobExecuting = scheduler.jobExecutingTable[jobPlan.Job.Name];jobExecuting{
		return
	}
	//构建执行状态信息
	jobExecuteInfo = BuildJobExecuteInfo(jobPlan)
	//保存执行状态
	scheduler.jobExecutingTable[jobPlan.Job.Name] = jobExecuteInfo

	//执行任务
	fmt.Println("执行任务:", jobExecuteInfo.Job.Name, jobExecuteInfo.PlanTime, jobExecuteInfo.RealTime)
	G_executor.ExecuteJob(jobExecuteInfo)
}
//推送任务变化事件
func (scheduler *Scheduler)PushJobEvent(jobEvent *JobEvent){
	scheduler.jobEventChan <- jobEvent
}
//推送任务执行结果事件
func(scheduler *Scheduler)PushJobResult(jobResult *JobExecuteResult){
	scheduler.jobResultChan <- jobResult
}
//处理任务事件
func (scheduler *Scheduler)handleJobEvent(jobEvent *JobEvent)  {
	var (
		err error
		jobSchedulePlan *JobSchedulePlan
		jobExisted bool
		jobExecuteInfo *JobExecuteInfo
		jobExecuting bool
	)

	switch jobEvent.EventType {
	case 1://保存任务事件
		if  jobSchedulePlan,err = BuildJobSchedulePlan(jobEvent.Job);err!=nil{
			return
		}
		scheduler.jobPlanTable[jobEvent.Job.Name] = jobSchedulePlan
	case 2://删除任务事件
		if jobSchedulePlan,jobExisted = scheduler.jobPlanTable[jobEvent.Job.Name];jobExisted{
			delete(scheduler.jobPlanTable,jobEvent.Job.Name)
		}
	case 3://强杀任务事件
		// 取消掉Command执行, 判断任务是否在执行中
		if jobExecuteInfo, jobExecuting = scheduler.jobExecutingTable[jobEvent.Job.Name]; jobExecuting {
			jobExecuteInfo.CancelFunc()	// 触发command杀死shell子进程, 任务得到退出
		}
	}
}
//处理任务执行结果(入mongo)
func (scheduler *Scheduler)handleJobResult(jobResult *JobExecuteResult){
	//删除任务执行状态
	delete(scheduler.jobExecutingTable,jobResult.ExecuteInfo.Job.Name)
	fmt.Println("任务执行完成：",jobResult.ExecuteInfo.Job.Name,string(jobResult.Output),jobResult.Err)
}

//初始化调度器
func InitScheduler()(err error){
	G_scheduler = &Scheduler{
		jobEventChan:make(chan *JobEvent,1000),
		jobPlanTable:make(map[string]*JobSchedulePlan),
		jobExecutingTable:make(map[string]*JobExecuteInfo),
		jobResultChan:make(chan *JobExecuteResult,1000),
	}
	//启动调度协程
	go G_scheduler.schedulerLoop()
	return
}
