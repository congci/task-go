package task

import (
	"log"
	"os/exec"
	"time"
	"unsafe"
)

type TaskTime struct {
	Timechannel
	Timewheel
}

var as = false

//任务执行池
type taskqueue struct {
	queue chan unsafe.Pointer
	num   int
}

//任务通用结构体
type Task struct {
	Tid        string   `json:"tid"`       //tid 任务唯一标志符、不能重复 //不取id避免和业务冲突
	Bid        string   `json:"bid"`       //可能是业务id
	ExTendTids []string `json:"extendtis"` //一个任务可能有附带的任务、关联的tid、可以一起删除等

	Extend interface{} `json:"extend"` // 扩展字段 任务具体任务可能需要

	TaskStr  string `json:"taskstr"` //server 接口的时候 存储任务task详情 json结构用于解析 -- 用户自定义任务 用于解析结构体、可以传这个、传str是为了方便看
	TaskByte []byte `json:"-"`       //任务执行过程的时候、减少转换解析

	Delay int64 `json:"delay"` //延时多少秒后开始执行、初次延时 或者 逻辑任务内延时标志、

	Num        uint `json:"num"`         //执行次数 用户状态观察
	SuccessNum uint `json:"success_num"` //执行成功次数
	FailNum    uint `json:"fail_num"`    //失败次数
	LimitNum   uint `json:"limit_num"`   //限制次数
	RetryNum   uint `json:"retry_num"`   //重试次数

	StartTime int64 `json:"starttime"` //每次任务开始时间
	EndTime   int64 `json:"endtime"`   //每次任务结束时间

	Duration int64 `json:"duration"` //时间间隔
	Cycle    int64 `json:"cycle"`    //如果是-1则永远不停、生命周期

	StartTaskTime int64 `json:"start_task_time"` //创建时间、如果没有则为当前时间\避免业务create_time 影响

	Command string `json:"common"` //如果有common代表是命令模式

	Func    func(*Task)        `json:"-"` //执行函数
	EndFunc func(*Task, Chanl) `json:"-"` //结束函数

	del int8 `json:"isdel"` //在timechannel是否已经被删除、如果延时再次加入的时候已经被删除、则不在加入
}

//类型统一结构
type TimeTask struct {
	T        *time.Ticker `json:"-"` //定时器channel
	TD       *time.Timer  `json:"-"` //延时器channel
	C        chan Chanl   `json:"-"` //用于通知任务close
	cyclenum int          //第几圈 时间轮
	index    int          //索引 时间轮
	*Task                 //存储的task
}

//任务信号
const (
	ADDTASK    = iota //0
	UPDATETASK        //1
	DELTASK           //2
	TIMEOUTASK        //3 自然过期
	DELAYTASK         //4 专门延时的信号 timechannel
)

//任务名称
const (
	TIMECHANNEL = iota
	TIMEWHEEL
)

//传输的结构体
type Chanl struct {
	Signal int
	Data   unsafe.Pointer
}

//返回的结构体
type TT interface {
	AddTc(task *Task) error
	UpdateTc(task *Task) error
	DelTc(id string) error
	GetAllTasks() []*Task
	Start()
}

var mode TT //初始化的时候会用

//工厂类
func NewTask(name int) TT {
	if name == TIMEWHEEL {
		mode = newTimeWheel()
		return mode
	} else if name == TIMECHANNEL {
		mode = newTimeChannel()
		return mode
	}
	return nil
}

//没有设定执行函数执行的函数
func runDo(t *Task) {
	log.Print("hello world")
}

func endDo(t *Task, stop interface{}) {
	log.Print("end", stop)
}

func do(t *TimeTask) {
	defer func() {
		if err := recover(); err != nil {
			log.Print(err)
			return
		}
	}()
	t.Num++
	//此处判断是否过期
	if t.Cycle != -1 && t.StartTaskTime+t.Cycle < time.Now().Unix() {
		if t.C != nil {
			t.C <- Chanl{Signal: TIMEOUTASK, Data: unsafe.Pointer(t)}
		}
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

	if t.Func != nil {
		t.Func(t.Task)
		//执行完、则自动加周期
		UpdateTimeClock(t.Task)
		return
	}
	runDo(t.Task)

}

//任务完成后操作
func end(t *Task, stop Chanl) {
	if t.EndFunc != nil {
		t.EndFunc(t, stop)
		return
	}
	endDo(t, stop)
}

//更新每次开始结束事件、时钟操作
func UpdateTimeClock(t *Task) {
	t.StartTime += t.Duration
	t.EndTime = t.StartTime + t.Duration
}

//任务成功
func SuccessTask(t *Task) {
	t.SuccessNum++
}

//任务失败
func FailTask(t *Task) {
	t.FailNum++
}

func autoUpdate(task *Task, v *Task) {
	//补充非外部变量
	if task.ExTendTids == nil && v.ExTendTids != nil {
		task.ExTendTids = v.ExTendTids
	}
	if task.Num == 0 && v.Num != 0 {
		task.Num = v.Num
	}
	if task.RetryNum == 0 && v.RetryNum != 0 {
		task.RetryNum = v.RetryNum
	}
	if task.StartTime == 0 && v.StartTime != 0 {
		task.StartTime = v.StartTime
	}
	if task.EndTime == 0 && v.EndTime != 0 {
		task.EndTime = v.EndTime
	}
	if task.FailNum == 0 && v.FailNum != 0 {
		task.FailNum = 0
	}
	if task.Func == nil && v.Func != nil {
		task.Func = v.Func
	}
	if task.EndFunc == nil && v.EndFunc != nil {
		task.EndFunc = v.EndFunc
	}
	if task.LimitNum == 0 && v.LimitNum != 0 {
		task.LimitNum = v.LimitNum
	}
	if task.TaskStr == "" && v.TaskStr != "" {
		task.TaskStr = v.TaskStr
	}
	if task.TaskByte == nil && v.TaskByte != nil {
		task.TaskByte = v.TaskByte
	}
	if task.SuccessNum == 0 && v.SuccessNum != 0 {
		task.SuccessNum = v.SuccessNum
	}
	if task.Cycle == 0 && v.Cycle != 0 {
		task.Cycle = v.Cycle
	}

	if task.Extend == nil && v.Extend != nil {
		task.Extend = v.Extend
	}
	if task.StartTaskTime == 0 && v.StartTaskTime != 0 {
		task.StartTaskTime = v.StartTaskTime
	}
	if task.Bid == "" && v.Bid != "" {
		task.Bid = v.Bid
	}
}

type Param struct {
}
