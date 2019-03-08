# task_go
>version 1.0.0

**methods*

- GetAllTasks 获取全部任务
- GetTasks 获取现有任务 不包括开始延后任务(非延时)
- AppendTask 加任务（这个是各个任务开始加载初始任务的时候调用、在执行过程中不能调用、）
- SetDefaultTaskName 设置默认任务名称
- SetDefaultTaskFunc 设置单个任务执行函数
- SetDefaultTaskEndFunc 设置单个任务结束函数
- SetAllDefaultTaskFunc 重置全部任务执行函数
- SetAllDefaultTaskEndFunc 重置全部任务结束函数
- Server 设置监听、可以接口加/更新/删除/获取状态 

- AddTc 添加任务 可执行过程中加 [注意、执行过程中慎用、会造成循环添加、]
- UpdateTc 修改任务
- DelTc 删除任务

> 另外支持 接口 增删改操作
```
    http.HandleFunc("/pushsub", addTc)      //增加任务
	http.HandleFunc("/updatesub", updateTc) //更新任务
	http.HandleFunc("/status", status)      //获取状态
	http.HandleFunc("/delsub", delTc)       //删除任务
```


> 多任务的实现：任务执行和结束函数都是用注入到map中来做的、用户可以设置默认任务名称、那个任务的执行函数和结束函数那么加入的任务就不需要设置taskName字段、系统会自动筛选出来执行

> 执行任务和结束任务没有的话会执行系统函数、输出hello world
> 自定义创建结构体 要包含 Task 结构体、其他的扩展字段可以自己扩展、加入的的task会存储到taskstr字段中保存、用户各个任务结构体解析和开发
> 必须有cycle、Duration、create_time 字段 create_time 格式是2006-01-02 15:04:05

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

	//设置初始任务 -- 里面可以调用AppendTask加载初始任务
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
**自定义结构体**

> 自定义结构体只要包含Task结构体既可
> Task 必须和自定义结构同级、不能加json
```

import (
	taskx "github.com/congci/task-go"
)

type Task = taskx.Task
type Topic struct{
	Task              
	Mustcon string `json:"mustcon"`
	MustNotCon string `json:"must_not_con"`
}
```