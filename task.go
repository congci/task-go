package kernel

import (
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

//执行榜单定时
type TimeTask struct {
	Id       int              `json:"id"`
	TaskName string           `json:"task_name"` //如果不传有默认
	T        *time.Ticker     `json:"-"`         //定时器channel
	TD       *time.Timer      `json:"-"`         //延时器channel
	C        chan interface{} `json:"-"`         //用于通知任务close
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

	StartTaskTime int64 `json:"starttasktime"` //任务开始时间
	Interrupted   int8  `json:"interrupted"`   //第一次导入的时候看是否中断过
}

var (
	//当前任务链
	Tc []*TimeTask

	//初始任务
	preTasks []*TimeTask

	//用于初始化的时候保存曾经中断的任务
	delayTc []*TimeTask

	//保存队列
	queue []*TimeTask
)

func Start() {
	//加载初始任务
	preloadTask()
	//异常 重试 //从队列中获取直接执行超过三次则失败
	go exception()
	//检测监控
	go check()
}

//获取全部task、包括开始中断中的
func GetAllTasks() []*TimeTask {
	RW.RLock()
	defer RW.RUnlock()
	return append(Tc, delayTc...)
}

//获取现有的task
func GetTasks() []*TimeTask {
	RW.RLock()
	defer RW.RUnlock()
	return Tc
}

//往任务链表里加数据
func AppendTask(t *TimeTask) {
	Tc = append(Tc, t)
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
func preloadTask() {
	//结束时间 - 当前时间 定时器执行、如果到了时间执行对应的操作、然后tick、保证接上以前的任务
	now := time.Now().Unix()
	for _, v := range preTasks {
		//如果任务已经结束、则忽略
		if v.Cycle != -1 && v.StartTaskTime+v.Cycle < now {
			continue
		}
		//任务中断过
		if (v.EndTime != 0 || v.Interrupted == 1) && (now > v.StartTime && v.EndTime > now && v.EndTime-now > 0 && v.EndTime-now < v.Duration) {
			log.Print("delay Tc" + v.TaskStr)
			delayTc = append(delayTc, v)
		} else {
			newTask(v)
		}
	}
	for _, v := range delayTc {
		func(v *TimeTask) {
			time.AfterFunc(time.Duration(v.EndTime-now)*time.Second, func() {
				RW.Lock()
				for k, value := range delayTc {
					if value.Id == v.Id {
						if k+1 > len(delayTc) {
							delayTc = append(delayTc[:k], delayTc[k+1:]...)
						} else {
							delayTc = delayTc[:k]
						}
					}
				}
				newTask(v)
				RW.Unlock()
				//到点了执行一次
				Do(v)
			})
		}(v)
	}
}

//延时任务
func newDelayTak(t *TimeTask) {
	ticker := time.NewTimer(time.Duration(t.Duration) * time.Second)
	c := make(chan interface{}, 1)
	t.TD = ticker
	t.C = c
	Tc = append(Tc, t)
	go func(t *TimeTask) {
		defer t.TD.Stop()
		for {
			select {
			case <-t.TD.C:
				//任务的task
				Do(t)
			case stop := <-t.C:
				End(t, stop)
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
	ticker := time.NewTicker(time.Duration(t.Duration) * time.Second)
	c := make(chan interface{}, 1)
	t.T = ticker
	t.C = c
	Tc = append(Tc, t)
	go func(t *TimeTask) {
		defer t.T.Stop()
		for {
			select {
			case <-t.T.C:
				//任务的task
				Do(t)
			case stop := <-t.C:
				End(t, stop)
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
	if t.TaskName != "" {
		RWFUNC.Lock()
		if funcx, ok := TaskFuncMap[t.TaskName]; ok {
			RWFUNC.RUnlock()
			funcx(t)
		} else {
			RWFUNC.RUnlock()
			runDo(t)
		}
	} else if DefaultTaskName != "" {
		RWFUNC.Lock()
		if funcx, ok := TaskFuncMap[DefaultTaskName]; ok {
			RWFUNC.RUnlock()
			funcx(t)
		} else {
			RWFUNC.RUnlock()
			runDo(t)
		}
		RWFUNC.RUnlock()
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
			RWFUNCEND.RUnlock()
			funcx(t, stop)
		} else {
			RWFUNCEND.RUnlock()
			endTask(t, stop)
		}
	} else if DefaultTaskName != "" {
		RWFUNCEND.Lock()
		if funcx, ok := TaskEndFuncMap[DefaultTaskName]; ok {
			RWFUNCEND.RUnlock()
			funcx(t, stop)
		} else {
			endTask(t, stop)
		}

	} else {
		endTask(t, stop)
	}
}
