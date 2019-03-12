package task

import (
	"container/list"
	"errors"
	"log"
	"os/exec"
	"sync"
	"time"
	"unsafe"
)

var (
	//默认间隔
	DefaultDuration int64 = 3600
)

//执行榜单定时
type TimeTask struct {
	T     *time.Ticker `json:"-"` //定时器channel
	TD    *time.Timer  `json:"-"` //延时器channel
	C     chan chanl   `json:"-"` //用于通知任务close
	*Task              //存储的task
}

//time + channel - 底层是最小堆
type Timechannel struct {
	T        *time.Ticker
	Intervel time.Duration
	tt       *list.List
	C        chan chanl //主线程通知信号
	//默认执行哪个任务
	defaulttaskname string
	taskfuncmap     map[string]func(*Task)
	taskendfuncmap  map[string]func(*Task, interface{})
	rwf             sync.RWMutex
	rwfend          sync.RWMutex
}

func newTimeChannel() *Timechannel {
	t := new(Timechannel)
	t.initT()
	return t
}

func (tc *Timechannel) initT() {
	tc.tt = list.New()
	tc.Intervel = 5                                  //暂时限定5
	tc.T = time.NewTicker(tc.Intervel * time.Second) //每几秒执行一次
	tc.taskfuncmap = make(map[string]func(*Task))
	tc.taskendfuncmap = make(map[string]func(*Task, interface{}))

}

//开始的时候加入
func (tc *Timechannel) LoadTasks() {
	if tc.tt.Len() == 0 {
		return
	}
	//结束时间 - 当前时间 定时器执行、如果到了时间执行对应的操作、然后tick、保证接上以前的任务
	now := time.Now().Unix()
	for e := tc.tt.Front(); e != nil; e = e.Next() {
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
		if v.EndTime != 0 && (now > v.StartTime && v.EndTime > now && v.EndTime-now > 0 && v.EndTime-now < v.Duration) {
			log.Print("delay Tc" + v.TaskStr)
			v.Interrupted = 1
		}
	}

	for e := tc.tt.Front(); e != nil; e = e.Next() {
		v := e.Value.(*TimeTask)
		if v.Interrupted == 1 {
			time.AfterFunc(time.Duration(v.EndTime-now)*time.Second, func() {
				tc.newTask(v)
				//到点了执行一次
				tc.do(v.Task)
			})
		} else {
			tc.newTask(v)
		}
	}
}

//开始 任务
func (tc *Timechannel) Start() {
	//默认死循环
	for {
		select {
		case <-tc.T.C:
			tc.check()
		case d := <-tc.C:
			//增加
			if d.signal == ADDTASK {
				tc.addTc((*Task)(d.data))
			}
			//更新
			if d.signal == UPDATETASK {
				tc.updateTc((*Task)(d.data))
			}
			//删除
			if d.signal == DELTASK {
				tc.delTc(*(*int)(d.data))
			}
		}
	}

}

//延时任务
func (tc *Timechannel) newDelayTak(t *TimeTask) {
	ticker := time.NewTimer(time.Duration(t.Duration) * time.Second)
	c := make(chan chanl, 1)
	t.TD = ticker
	t.C = c
	go func(t *TimeTask) {
		defer t.TD.Stop()
		for {
			select {
			case <-t.TD.C:
				//任务的task
				tc.do(t.Task)
			case stop := <-t.C:
				if DELTASK == stop.signal {
					return
				}
				tc.end(t.Task, stop)
				return
			}
		}
	}(t)
}

//定时任务
func (tc *Timechannel) newTask(t *TimeTask) {
	log.Println(t)
	if t.Duration == 0 {
		t.Duration = DefaultDuration
	}
	if t.Delay != 0 {
		tc.newDelayTak(t)
		return
	}

	t.Interrupted = 0
	ticker := time.NewTicker(time.Duration(t.Duration) * time.Second)
	c := make(chan chanl, 1)
	t.T = ticker
	t.C = c
	go func(t *TimeTask) {
		defer t.T.Stop()
		for {
			select {
			case <-t.T.C:
				//任务的task
				tc.do(t.Task)
			case stop := <-t.C:
				if stop.signal == UPDATETASK {
					return
				}
				tc.end(t.Task, stop)
				return
			}
		}
	}(t)
}

//执行任务
func (tc *Timechannel) do(t *Task) {
	defer func() {
		if err := recover(); err != nil {
			log.Print(err)
			return
		}
	}()
	t.Num++
	//加个保险
	if t.StartTaskTime+t.Cycle < time.Now().Unix() {
		tc.end(t, TIMEOUTASK) //过期
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
		tc.rwf.Lock()
		if funcx, ok := tc.taskfuncmap[t.TaskName]; ok {
			tc.rwf.Unlock()
			funcx(t)
		} else {
			tc.rwf.Unlock()
			runDo(t)
		}
	} else if tc.defaulttaskname != "" {
		tc.rwf.Lock()
		if funcx, ok := tc.taskfuncmap[tc.defaulttaskname]; ok {
			tc.rwf.Unlock()
			funcx(t)
		} else {
			tc.rwf.Unlock()
			runDo(t)
		}
	} else {
		runDo(t)
	}
}

func runDo(t *Task) {
	log.Print("hello world")
}

//更新每次开始结束事件、时钟操作
func (t *TimeTask) UpdateTimeClock() {
	sn := time.Now().Unix()
	enn := sn + t.Duration
	t.StartTime = sn
	t.EndTime = enn
}

//任务成功
func (t *TimeTask) SuccessTask() {
	t.SuccessNum++
}

//任务失败
func (t *TimeTask) FailTask() {
	t.FailNum++
}
func endTask(t *Task, stop interface{}) {
	log.Print("end", stop)
}

//任务完成后操作
func (tc *Timechannel) end(t *Task, stop interface{}) {
	if t.TaskName != "" {
		tc.rwfend.Lock()
		if funcx, ok := tc.taskendfuncmap[t.TaskName]; ok {
			tc.rwfend.Unlock()
			funcx(t, stop)
		} else {
			tc.rwfend.Unlock()
			endTask(t, stop)
		}
	} else if tc.defaulttaskname != "" {
		tc.rwfend.Lock()
		if funcx, ok := tc.taskendfuncmap[tc.defaulttaskname]; ok {
			tc.rwfend.Unlock()
			funcx(t, stop)
		} else {
			tc.rwfend.Unlock()
			endTask(t, stop)
		}

	} else {
		endTask(t, stop)
	}
}

//每秒扫描一次、检查数据是否过期
func (tc *Timechannel) check() {
	now := time.Now().Unix()
	for e := tc.tt.Front(); e != nil; {
		t := e.Value.(*TimeTask)
		if (t.Cycle != -1 && t.StartTaskTime+t.Cycle < now) || (t.LimitNum != 0 && t.Num > t.LimitNum) {
			t.C <- chanl{signal: TIMEOUTASK}
			tc.tt.Remove(e)
		}
		e = e.Next()
	}
}

//=== 外部调用

//只在加载前操作、因此没有冲突
func (tc *Timechannel) AddOnlyTask(t *Task) {
	var tmp TimeTask
	tmp.Task = t
	tc.tt.PushBack(&tmp)
}

//设置默认task
func (tc *Timechannel) SetDefaultTaskName(name string) {
	tc.defaulttaskname = name
}

//设置任务执行函数
func (tc *Timechannel) SetDefaultTaskFunc(name string, f func(*Task)) {
	tc.taskfuncmap[name] = f
}

//设置任务执行函数\不需要、必须在任务开始前设置
func (tc *Timechannel) SetDefaultTaskEndFunc(name string, f func(*Task, interface{})) {
	tc.taskendfuncmap[name] = f
}

//设置任务执行函数
func (tc *Timechannel) SetAllDefaultTaskFunc(fs map[string]func(*Task)) {
	tc.taskfuncmap = fs
}

//设置任务执行函数
func (tc *Timechannel) SetAllDefaultTaskEndFunc(fs map[string]func(*Task, interface{})) {
	tc.taskendfuncmap = fs
}

//更新
func (tc *Timechannel) UpdateTc(task *Task) error {
	tc.C <- chanl{signal: DELTASK, data: unsafe.Pointer(task)}
	return nil
}

//更新任务
func (tc *Timechannel) updateTc(task *Task) error {
	for e := tc.tt.Front(); e != nil; {
		v := e.Value.(*TimeTask)
		if v.Id == task.Id {
			tc.tt.Remove(e)
			v.C <- chanl{signal: TIMEOUTASK}
			tc.addTc(task)
			return nil
		}
		e = e.Next()
	}
	//重开新的定时器
	return errors.New("fail")
}

//添加
func (tc *Timechannel) AddTc(task *Task) error {
	tc.C <- chanl{signal: ADDTASK, data: unsafe.Pointer(task)}
	return nil
}

//往任务链表里加数据
func (tc *Timechannel) addTc(t *Task) error {
	var tmp TimeTask
	tmp.Task = t
	tc.tt.PushBack(&tmp)
	tc.newTask(&tmp)
	return nil
}

//删除任务
func (tc *Timechannel) DelTc(id int) error {
	tc.C <- chanl{signal: DELTASK, data: unsafe.Pointer(&id)}
	return nil
}

//
//删除任务
func (tc *Timechannel) delTc(id int) error {
	for e := tc.tt.Front(); e != nil; {
		v := e.Value.(*TimeTask)
		if v.Id == id {
			tc.tt.Remove(e)
			v.C <- chanl{signal: TIMEOUTASK}
			return nil
		}
		e = e.Next()
	}
	return errors.New("del task fail")
}

//获取全部task、包括开始中断中的
func (tc *Timechannel) GetAllTasks() []*Task {
	var ts []*Task
	for iter := tc.tt.Back(); iter != nil; iter = iter.Prev() {
		v := iter.Value.(*TimeTask)
		ts = append(ts, v.Task)
	}
	return ts
}
