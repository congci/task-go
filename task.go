package task

import (
	"os/exec"
	"sync"
	"time"

	"log"
)

var RW sync.RWMutex
var RWD sync.RWMutex
var RWFUNC sync.RWMutex
var RWFUNCEND sync.RWMutex

var (
	//默认间隔
	DefaultDuration int64 = 3600
	//默认执行哪个任务
	DefaultTaskName string
	TaskFuncMap     = make(map[string]func(*TimeTask))
	TaskEndFuncMap  = make(map[string]func(*TimeTask, interface{}))
)

const (
	TIMEOUTASK = 2 //自然过期
	RENEWTASK  = 3 //代表新开一个
	DELTASK    = 9 //删除
)

//执行榜单定时
type TimeTask struct {
	T         *time.Ticker     `json:"-"` //定时器channel
	TD        *time.Timer      `json:"-"` //延时器channel
	C         chan interface{} `json:"-"` //用于通知任务close
	Cycle_Num int64            `json:"cycle_num"`
	Task
}

//任务通用结构体
type Task struct {
	Id       int    `json:"id"`        //id 不能重复
	TaskName string `json:"task_name"` //如果不传有默认
	TaskStr  string `json:"taskstr"`   //存储任务task详情 json结构用于解析 -- 用户自定义任务 用于解析结构体、必须
	Delay    int    `json:"delay"`     //如果有延时、则代表任务是一次性任务

	Num        uint `json:"num"`         //执行次数 用户状态观察
	SuccessNum uint `json:"success_num"` //执行成功次数
	FailNum    uint `json:"fail_num"`    //失败次数
	LimitNum   uint `json:"limit_num"`   //限制次数
	RetryNum   uint `json:"retry_num"`   //重试次数

	StartTime int64 `json:"starttime"` //每次任务开始时间
	EndTime   int64 `json:"endtime"`   //每次任务结束时间

	Duration int64 `json:"duration"` //时间间隔
	Cycle    int64 `json:"cycle"`    //如果是-1则永远不停、生命周期

	Create_time string `json:"create_time"` //创建时间、如果没有则为当前时间
	Update_time string `json:"update_time"` //更新时间

	Command       string `json:"common"`        //如果有common代表是命令模式
	StartTaskTime int64  `json:"starttasktime"` //任务开始时间
	Interrupted   int8   `json:"interrupted"`   //第一次导入的时候看是否中断过
}

var tc = NewQueue()
var queue = NewQueue()

func Start() {
	GO()
	//异常 重试 //从队列中获取直接执行超过三次则失败
	go exception()
	//检测监控
	go check()
}

//获取全部task、包括开始中断中的
func GetAllTasks() []*TimeTask {
	var ts []*TimeTask
	for iter := tc.data.Back(); iter != nil; iter = iter.Prev() {
		v := iter.Value.(*TimeTask)
		ts = append(ts, v)
	}
	return ts
}

//往任务链表里加数据
func AppendTask(t *TimeTask) {
	tc.Push(t)
}

//设置默认task
func SetDefaultTaskName(name string) {
	DefaultTaskName = name
}

//设置任务执行函数
func SetDefaultTaskFunc(name string, f func(*TimeTask)) {
	RWFUNC.Lock()
	TaskFuncMap[name] = f
	RWFUNC.Unlock()
}

//设置任务执行函数
func SetDefaultTaskEndFunc(name string, f func(*TimeTask, interface{})) {
	RWFUNCEND.Lock()
	TaskEndFuncMap[name] = f
	RWFUNCEND.Unlock()
}

//设置任务执行函数
func SetAllDefaultTaskFunc(fs map[string]func(*TimeTask)) {
	TaskFuncMap = fs
}

//设置任务执行函数
func SetAllDefaultTaskEndFunc(fs map[string]func(*TimeTask, interface{})) {
	TaskEndFuncMap = fs
}

//初始化 任务
func GO() {
	//结束时间 - 当前时间 定时器执行、如果到了时间执行对应的操作、然后tick、保证接上以前的任务
	now := time.Now().Unix()
	for e := tc.data.Front(); e != nil; e = e.Next() {
		v := e.Value.(*TimeTask)
		//如果任务已经结束、则忽略
		if v.Cycle != -1 && v.StartTaskTime+v.Cycle < now {
			continue
		}
		//代表这个任务记录有问题、1\结束时间小于现在 2、开始时间小于 最近一个周期
		if (v.EndTime != 0 && v.EndTime < now) || (v.StartTime != 0 && v.StartTime < now-v.Duration) {
			v.StartTime = now
			v.EndTime = now + v.Duration
		}
		//任务中断过
		if (v.EndTime != 0 || v.Interrupted == 1) && (now > v.StartTime && v.EndTime > now && v.EndTime-now > 0 && v.EndTime-now < v.Duration) {
			log.Print("delay Tc" + v.TaskStr)
			v.Interrupted = 1
		}
	}

	for e := tc.data.Front(); e != nil; e = e.Next() {
		v := e.Value.(*TimeTask)
		if v.Interrupted == 1 {
			time.AfterFunc(time.Duration(v.EndTime-now)*time.Second, func() {
				newTask(v)
				//到点了执行一次
				Do(v)
			})
		} else {
			newTask(v)
		}
	}
}

//延时任务
func newDelayTak(t *TimeTask) {
	ticker := time.NewTimer(time.Duration(t.Duration) * time.Second)
	c := make(chan interface{}, 1)
	t.TD = ticker
	t.C = c
	go func(t *TimeTask) {
		defer t.TD.Stop()
		for {
			select {
			case <-t.TD.C:
				//任务的task
				Do(t)
			case stop := <-t.C:
				if stop == RENEWTASK {
					return
				}
				End(t, stop)
				return
			}
		}
	}(t)
}

//定时任务
func newTask(t *TimeTask) {
	log.Println(t)
	if t.Duration == 0 {
		t.Duration = DefaultDuration
	}
	if t.Delay != 0 {
		newDelayTak(t)
		return
	}

	t.Interrupted = 0
	ticker := time.NewTicker(time.Duration(t.Duration) * time.Second)
	c := make(chan interface{}, 1)
	t.T = ticker
	t.C = c
	go func(t *TimeTask) {
		defer t.T.Stop()
		for {
			select {
			case <-t.T.C:
				//任务的task
				Do(t)
			case stop := <-t.C:
				if stop == RENEWTASK {
					return
				}
				End(t, stop)
				return
			}
		}
	}(t)
}

//执行任务
func Do(t *TimeTask) {
	defer func() {
		if err := recover(); err != nil {
			log.Print(err)
			return
		}
	}()
	t.Num++
	//加个保险
	if t.StartTaskTime+t.Cycle < time.Now().Unix() {
		return
	}
	if t.Command != "" {
		cmd := exec.Command(t.Command)
		if _, err := cmd.Output(); err != nil {
			log.Println(err)
			t.FailNum++

		} else {
			t.SuccessNum++
		}
	}

	if t.TaskName != "" {
		RWFUNC.Lock()
		if funcx, ok := TaskFuncMap[t.TaskName]; ok {
			RWFUNC.Unlock()
			funcx(t)
		} else {
			RWFUNC.Unlock()
			runDo(t)
		}
	} else if DefaultTaskName != "" {
		RWFUNC.Lock()
		if funcx, ok := TaskFuncMap[DefaultTaskName]; ok {
			RWFUNC.Unlock()
			funcx(t)
		} else {
			RWFUNC.Unlock()
			runDo(t)
		}
	} else {
		runDo(t)
	}
}

func runDo(t *TimeTask) {
	log.Print("hello world")
}

//更新每次开始结束事件、时钟操作
func UpdateTimeClock(t *TimeTask) {
	sn := time.Now().Unix()
	enn := sn + t.Duration
	t.StartTime = sn
	t.EndTime = enn
}

//任务成功
func SuccessTask(t *TimeTask) {
	t.SuccessNum++
}

//任务失败
func FailTask(t *TimeTask) {
	t.FailNum++
}
func endTask(t *TimeTask, stop interface{}) {
	log.Print("end", stop)
}

//任务完成后操作
func End(t *TimeTask, stop interface{}) {
	if t.TaskName != "" {
		RWFUNCEND.Lock()
		if funcx, ok := TaskEndFuncMap[t.TaskName]; ok {
			RWFUNCEND.Unlock()
			funcx(t, stop)
		} else {
			RWFUNCEND.Unlock()
			endTask(t, stop)
		}
	} else if DefaultTaskName != "" {
		RWFUNCEND.Lock()
		if funcx, ok := TaskEndFuncMap[DefaultTaskName]; ok {
			RWFUNCEND.Unlock()
			funcx(t, stop)
		} else {
			RWFUNCEND.Unlock()
			endTask(t, stop)
		}

	} else {
		endTask(t, stop)
	}
}
