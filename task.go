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

//任务通用结构体
type Task struct {
	Id       int         `json:"id"`        //id 不能重复
	TaskName string      `json:"task_name"` //如果不传有默认
	TaskStr  string      `json:"taskstr"`   //server 接口的时候 存储任务task详情 json结构用于解析 -- 用户自定义任务 用于解析结构体、必须
	Extend   interface{} // 扩展字段 任务具体任务可能需要
	Delay    int         `json:"delay"` //如果有延时、则代表任务是一次性任务

	Num        uint `json:"num"`         //执行次数 用户状态观察
	SuccessNum uint `json:"success_num"` //执行成功次数
	FailNum    uint `json:"fail_num"`    //失败次数
	LimitNum   uint `json:"limit_num"`   //限制次数
	RetryNum   uint `json:"retry_num"`   //重试次数

	StartTime int64 `json:"starttime"` //每次任务开始时间
	EndTime   int64 `json:"endtime"`   //每次任务结束时间

	Duration int64 `json:"duration"` //时间间隔
	Cycle    int64 `json:"cycle"`    //如果是-1则永远不停、生命周期

	Create_time int64 `json:"create_time"` //创建时间、如果没有则为当前时间
	Update_time int64 `json:"update_time"` //更新时间

	Command string `json:"common"` //如果有common代表是命令模式

	Func    func(*Task)        //执行函数
	EndFunc func(*Task, Chanl) //结束函数

	StartTaskTime int64 `json:"starttasktime"` //任务开始时间

	Interrupted int8 `json:"interrupted"` //第一次导入的时候看是否中断过
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
	DelTc(id int) error
	GetAllTasks() []*Task
	AddOnlyTask(task *Task)
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
			t.C <- Chanl{Signal: TIMEOUTASK}
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
	sn := time.Now().Unix()
	enn := sn + t.Duration
	t.StartTime = sn
	t.EndTime = enn
}

//任务成功
func SuccessTask(t *Task) {
	t.SuccessNum++
}

//任务失败
func FailTask(t *Task) {
	t.FailNum++
}

func formatTask(task *Task) {
	//根据条件设置对应的func
	now := time.Now().Unix()
	if task.Create_time == 0 {
		task.StartTaskTime = now
	}
	if task.StartTaskTime == 0 {
		task.StartTaskTime = now
	}

	if task.StartTime == 0 {
		task.StartTime = now
	}
	if task.EndTime == 0 {
		task.EndTime = now + task.Duration
	}
}
