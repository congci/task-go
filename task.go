package task

import (
	"unsafe"
)

type TaskTime struct {
	Timechannel
	Timewheel
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

	Command string `json:"common"` //如果有common代表是命令模式

	StartTaskTime int64 `json:"starttasktime"` //任务开始时间

	Interrupted int8 `json:"interrupted"` //第一次导入的时候看是否中断过
}

//任务信号
const (
	ADDTASK = iota
	UPDATETASK
	DELTASK
	TIMEOUTASK //自然过期
)

//任务名称
const (
	TIMECHANNEL = iota
	TIMEWHEEL
)

//传输的结构体
type chanl struct {
	signal int
	data   unsafe.Pointer
}

//返回的结构体
type TT interface {
	AddTc(task *Task) error
	UpdateTc(task *Task) error
	DelTc(id int) error
	GetAllTasks() []*Task
	LoadTasks()
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
