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


