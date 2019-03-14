# task_go
>version 2.0.0
> 时间轮 循环时间轮上的槽、如果cycle_num 为0 则代表过期执行、cycle_num 代表执行
>  默认定时器、其实golang底层是最小堆 + channel实现的\最小堆上注册、然后每次取最小值、然后sleep、然后发送channel、
> 最小堆
> 选择方式 

> 任务需要是task结构体 然后可以有扩展字段\函数中实现一般是把自己定义的对象转string、然后放到Taskstr字段里、任务里自己解析



**methods*
> 加的函数
- AddTc 添加任务
- UpdateTc 修改任务
- DelTc 删除任务
- GetAllTasks 获取全部任务  **任务小的时候可以用、任务大了就不要用了、这个是循环所有的任务**

> 建议支持接口 增删改操作 需要自己实现对应task的执行赋值
> 可以根据example例子目录中来写 

>example 例子中的支持接口

```
    http.HandleFunc("/pushsub", addTc)      //增加任务
	http.HandleFunc("/updatesub", updateTc) //更新任务
	http.HandleFunc("/status", status)      //获取状态
	http.HandleFunc("/delsub", delTc)       //删除任务
```


**增加删除都是在 主线程、因此不需要加锁、分发的协程仅仅是执行**
**任务开始前加是直接在链表中加、任务开始后加任务、则会通过channel通知主协程加**
**唯一注意的就是不能在任务未开始的时候、用协程加任务、加任务只能在任务未开始主协程中加、或者任务开始后、如果任务开始在另一个协程序中、里面又有耗时情况、则可能会造成线程不安全**

> 必须有cycle、Duration


```
 type Task = task.Task
 type Chanl = task.Chanl
 var mode task.TT //这个是类型（时间轮/timer定时器）实现的接口

 //初始化
mode = task.NewTask(task.TIMEWHEEL)
```



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


**注意**
>现在的golang默认的定时器实现是不适合大规模任务的、golang底层虽然会复用已经创建的协程、但是还是有撑爆内存的风险、可以自己实现协程池、自己实现超时删除等逻辑