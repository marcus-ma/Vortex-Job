package Vortex_Job

import (
	"os/exec"
	"time"
)

// 任务执行器
type Executor struct {}

var (G_executor *Executor)

// 执行一个任务
func (executor *Executor) ExecuteJob(info *JobExecuteInfo) {
	go func() {
		var (
			cmd *exec.Cmd
			err error
			output []byte
			result *JobExecuteResult
		)
		// 任务结果
		result = &JobExecuteResult{
			ExecuteInfo: info,
			Output: make([]byte, 0),
		}

		// 记录任务开始时间
		result.StartTime = time.Now()

		// 执行shell命令
		cmd = exec.CommandContext(info.CancelCtx, G_config.BashDir, "-c", info.Job.Command)
		// 执行并捕获输出
		output, err = cmd.CombinedOutput()

		// 记录任务结束时间
		result.EndTime = time.Now()
		//任务返回结果
		result.Output = output
		//任务错误信息
		result.Err = err

		// 任务执行完成后，把执行的结果返回给Scheduler，Scheduler会从executingTable中删除掉执行记录
		G_scheduler.PushJobResult(result)

	}()
}

//  初始化执行器
func InitExecutor() (err error) {
	G_executor = &Executor{}
	return
}
