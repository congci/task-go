# task_go
>version 1.0.0

**methods*

- GetAllTasks 获取全部任务
- GetTasks 获取现有任务 不包括开始延后任务(非延时)
- AppendTask 加任务 
- SetDefaultTaskName 设置默认任务名称
- SetDefaultTaskFunc 设置单个任务执行函数
- SetDefaultTaskEndFunc 设置单个任务结束函数
- SetAllDefaultTaskFunc 重置全部任务执行函数
- SetAllDefaultTaskEndFunc 重置全部任务结束函数
- Server 设置监听、可以接口加/更新/删除/获取状态 

```
    http.HandleFunc("/pushsub", addTc)      // 增加任务
	http.HandleFunc("/updatesub", updateTc) //更新任务
	http.HandleFunc("/status", status)      //获取状态
	http.HandleFunc("/delsub", delTc)       //删除任务
```


> 多任务的实现：任务执行和结束函数都是用注入到map中来做的、用户可以设置默认任务名称、那个任务的执行函数和结束函数那么加入的任务就不需要设置taskName字段、系统会自动筛选出来执行

> 执行任务和结束任务没有的话会执行系统函数、输出hello world
> 自定义创建结构体 要包含 Task 结构体
> 必须有cycle、Duration、create_time 字段 create_time 格式是2006-01-02 15:04:05
> 加入的的task会存储到taskstr字段中保存、用户各个任务结构体解析和开发

**eg**
```
package main

import (
	"flag"
	"opinion-monitor/cmd/subscript/tasks/topic"

	taskx "github.com/congci/task-go"
)

var port = flag.String("port", "0.0.0.0:8001", "port set")

func main() {
	flag.Parse()

	//设置初始任务
	topic.Preload()
	//设置topic任务的函数
	taskx.SetDefaultTaskName("topic")
	taskx.SetDefaultTaskFunc("topic", topic.RunDo)
	taskx.SetDefaultTaskEndFunc("topic", topic.ChangeStatus)
	
	taskx.Start()

	taskx.Server(*port)
}
```


**建议目录结构**
- cmd
- - tasks 
  - - topic 
  - - - topic.go
  - - - email.go
    
- - conf.go
- - main.go

> 结构体可以直接用类型别名  
```
 type TimeTask = taskx.TimeTask
```
**类型结构体**

```
//执行榜单定时
type TimeTask struct {
	Id       int    `json:"id"`
	TaskName string `json:"task_name"` //如果不传有默认

	T  *time.Ticker     `json:"-"` //定时器channel
	TD *time.Timer      `json:"-"` //延时器channel
	C  chan interface{} `json:"-"` //用于通知任务close
	Task
}

//任务通用结构体
type Task struct {
	TaskStr    string `json:"taskstr"`     //存储任务task详情 json结构用于解析 -- 用户自定义任务 用于解析结构体、必须
	Delay      int    `json:"delay"`       //如果有延时、则代表任务是一次性任务
	Num        uint   `json:"num"`         //执行次数 用户状态观察
	SuccessNum uint   `json:"success_num"` //执行成功次数
	FailNum    uint   `json:"fail_num"`    //失败次数
	LimitNum   uint   `json:"limit_num"`
	RetryNum   uint   `json:"retry_num"` //重试次数
	StartTime  int64  `json:"starttime"` //每次任务开始时间
	EndTime    int64  `json:"endtime"`   //每次任务结束时间
	Duration   int64  `json:"duration"`  //时间间隔
	Cycle      int64  `json:"cycle"`     //如果是-1则永远不停、生命周期

	Create_time string `json:"create_time"` //创建时间、如果没有则为当前时间
	Update_time string `json:"update_time"` //更新时间

	Command       string `json:"common"`        //如果有common代表是命令模式
	StartTaskTime int64  `json:"starttasktime"` //任务开始时间
	Interrupted   int8   `json:"interrupted"`   //第一次导入的时候看是否中断过
}
```