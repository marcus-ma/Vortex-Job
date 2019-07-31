package Vortex_Job

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"regexp"
	"time"
)

// 任务管理器
type JobMgr struct {
	client *mongo.Client
	collection *mongo.Collection
}


var (G_jobMgr *JobMgr)

type JobRecord struct {
	JobId_ string `json:"job_id,omitempty" bson:"_id,omitempty"`//任务ID，mongo的主键ID
	JobName string `json:"job_name" bson:"job_name"`//任务名
	JobInfo Job  `json:"job_info" bson:"job_info"`//任务具体信息
	JobStatus int32 `json:"job_status" bson:"job_status"`//任务状态(是否在执行)
	JobCreateTime int64 `json:"job_create_time" bson:"job_create_time"`//任务创建时间
}

//校验shell是否合法(rm或者rm -rf为不合法)
func checkCommand(data []byte) bool {
	re,_ := regexp.Compile(`,"command":".*?(;|rm -rf|rm).*?",`)
	ret := re.FindStringSubmatch(string(data))
	if len(ret)!=0{return false}
	return true
}

//创建任务
func (jobMgr *JobMgr) addJob(data []byte) (err error){
	var(
		job *Job
		isLegal bool
		)

	if isLegal = checkCommand(data);!isLegal{
		return errors.New("shell不合法，包含敏感命令(rm、rm -rf等)")
	}

	//反序列job
	if job,err = UnpackJob(data);err!=nil{
		return
	}

	//插入mongo
	if _,err = jobMgr.collection.InsertOne(context.TODO(),&JobRecord{
		JobName:job.Name,
		JobInfo:Job{
			CronExpr:job.CronExpr,
			Command:job.Command,
		},
		JobStatus:1,
		JobCreateTime:time.Now().Unix(),
	});err!=nil{
		return
	}

	//构建一个创建event
	G_scheduler.PushJobEvent(BuildJobEvent(1,job))

	return nil
}

//停止任务
func (jobMgr *JobMgr) killJob(data []byte)(err error) {

	re,_ := regexp.Compile(`{"name":"(.*?)"`)
	ret := re.FindStringSubmatch(string(data))

	//更改mongo中任务状态
	if _,err = jobMgr.collection.UpdateOne(context.TODO(),bson.M{"job_name":ret[1]},bson.M{"$set":bson.M{"job_status":0}});err!=nil{
		return
	}

	//构建一个创建event
	//删除任务
	G_scheduler.PushJobEvent(BuildJobEvent(2,&Job{Name:ret[1]}))
	//结束任务进程
	G_scheduler.PushJobEvent(BuildJobEvent(3,&Job{Name:ret[1]}))

	return nil
}

//重启任务
func (jobMgr *JobMgr) rebootJob(data []byte)(err error) {
	var(
		record JobRecord
		job *Job
	)

	re,_ := regexp.Compile(`{"name":"(.*?)"`)
	ret := re.FindStringSubmatch(string(data))


	//更改mongo中任务状态
	if err = jobMgr.collection.FindOneAndUpdate(context.TODO(),bson.D{{"job_name",ret[1]}},bson.M{"$set":bson.M{"job_status":1}}).Decode(&record);err!=nil{
		return
	}
	job = &Job{
		Name:record.JobName,
		Command:record.JobInfo.Command,
		CronExpr:record.JobInfo.CronExpr,
	}
	//构建一个创建event
	G_scheduler.PushJobEvent(BuildJobEvent(1,job))

	return nil
}

//列出任务
func (jobMgr *JobMgr) listJob()(box []interface{},err error)  {
	var (
		cursor *mongo.Cursor
		record *JobRecord
	)

	//从mongo中获取任务
	if cursor,err = jobMgr.collection.Find(context.TODO(),bson.M{});err!=nil{
		return
	}
	//释放游标
	defer cursor.Close(context.TODO())

	//遍历结果集
	for cursor.Next(context.TODO()){
		var item =  bson.D{}
		//反序列化bson到struct
		if err = cursor.Decode(&item);err!=nil{
			continue
		}
		//定义一个日志对象
		record = &JobRecord{
			JobId_:item[0].Value.(primitive.ObjectID).Hex(),
			JobName:item[1].Value.(string),
			JobInfo:Job{
				Command:item[2].Value.(bson.D)[0].Value.(string),
				CronExpr:item[2].Value.(bson.D)[1].Value.(string),
			},
			JobStatus:item[3].Value.(int32),
			JobCreateTime:item[4].Value.(int64),
		}
		box = append(box,record)
	}

	return box,nil
}

//任务的修改
func (jobMgr *JobMgr) modifyJob(data []byte) (err error){
	var(
		job *Job
		isLegal bool
		)

	if isLegal = checkCommand(data);!isLegal{
		return errors.New("shell不合法，包含敏感命令(rm、rm -rf等)")
	}

	//获取任务的id
	re,_ := regexp.Compile(`{"id":"(.*?)"`)
	ret := re.FindStringSubmatch(string(data))
	objectId,_ := primitive.ObjectIDFromHex(ret[1])

	//获取任务json
	re,_ = regexp.Compile(`"id":"(.*?)",`)
	data = []byte(re.ReplaceAllString(string(data),""))
	//反序列job
	if job,err = UnpackJob(data);err!=nil{
		return
	}
	//修改mongo
	var item =  bson.D{}
	if err = jobMgr.collection.FindOneAndUpdate(context.TODO(),
		bson.M{"_id":objectId},
		bson.M{"$set":bson.M{
			"job_name":job.Name,
			"job_info.command":job.Command,
			"job_info.cron_expr":job.CronExpr,}}).Decode(&item);err!=nil{
			return
		}

	//构建一个创建event
	//删除任务
	G_scheduler.PushJobEvent(BuildJobEvent(2,&Job{Name:item[1].Value.(string)}))
	//结束任务进程
	G_scheduler.PushJobEvent(BuildJobEvent(3,&Job{Name:item[1].Value.(string)}))
	//重新投递任务
	G_scheduler.PushJobEvent(BuildJobEvent(1,job))

	return
}




// 初始化管理器
func InitJobMgr() (err error) {
	 var(
		 ctx context.Context
		 client *mongo.Client
		 collection *mongo.Collection
		 cursor *mongo.Cursor
		 record *JobRecord
	 )

	//1.建立连接
	ctx,_ = context.WithTimeout(
		context.TODO(),
		time.Duration(G_config.MongodbConnectTimeout)*time.Millisecond,
		)
	if client,err =
		mongo.Connect(ctx,options.Client().ApplyURI(G_config.MongodbUri));err!=nil{
		return
	}
	//2.选择数据库和表collection
	collection = client.Database("cron").Collection("jobs")

	// 赋值单例
	G_jobMgr = &JobMgr{
		client: client,
		collection:collection,
	}

	//从mongo中获取当前任务来执行
	if cursor,err = collection.Find(context.TODO(),bson.M{"job_status":1});err!=nil{
		return
	}

	//遍历结果集
	for cursor.Next(context.TODO()){
		//定义一个日志对象
		record = &JobRecord{}
		//反序列化bson到struct
		if err = cursor.Decode(record);err!=nil{
			continue
		}

		//跳过不合法命令
		re,_ := regexp.Compile(`(;|rm -rf|rm)`)
		ret := re.FindStringSubmatch(record.JobInfo.Command)
		if len(ret)!=0{continue}

		//同步给scheduler(调度协程)
		G_scheduler.PushJobEvent(BuildJobEvent(1,&Job{
			Name:record.JobName,
			Command:record.JobInfo.Command,
			CronExpr:record.JobInfo.CronExpr,
		}))
	}
	//释放游标
	cursor.Close(context.TODO())

	return
}