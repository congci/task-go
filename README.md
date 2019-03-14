# task_go
>version 2.0.0
> 时间轮 循环时间轮上的槽、如果cycle_num 为0 则代表过期执行、cycle_num 代表执行
>  默认定时器、其实golang底层是最小堆 + channel实现的\最小堆上注册、然后每次取最小值、然后sleep、然后发送channel、
> 最小堆
> 选择方式 task.NewTask()

**methods*


> 加的函数
- AddOnlyTask 加任务（只加入准备队列、不会执行）
- AddTc 添加任务
- UpdateTc 修改任务
- DelTc 删除任务
- GetAllTasks 获取全部任务

> 建议支持接口 增删改操作 需要自己实现对应task的执行赋值

```
    http.HandleFunc("/pushsub", addTc)      //增加任务
	http.HandleFunc("/updatesub", updateTc) //更新任务
	http.HandleFunc("/status", status)      //获取状态
	http.HandleFunc("/delsub", delTc)       //删除任务
```


> 必须有cycle、Duration、create_time 字段 create_time 格式是2006-01-02 15:04:05

**eg**
```
//任务名称
const (
	TIMECHANNEL = iota  //默认的
	TIMEWHEEL //时间轮
)

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
 type TimeTask = task.Task
```

> 任务需要是task结构体 然后可以有扩展字段

> 增加删除都是在 主线程、因此不需要加锁、分发的协程仅仅是执行