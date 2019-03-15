package task

import (
	"container/list"
	"errors"
	"log"
	"time"
	"unsafe"

	"github.com/rs/xid"
)

var (
	//默认间隔
	DefaultDuration int64 = 3600
)

//time + channel - 底层是最小堆
type Timechannel struct {
	T        *time.Ticker
	Intervel time.Duration
	tt       *list.List
	C        chan Chanl //主线程通知信号
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
	tc.C = make(chan Chanl, 1)
}

//如果有有开始任务、那么执行
func (tc *Timechannel) CheckAndTask(t *TimeTask) {
	//结束时间 - 当前时间 定时器执行、如果到了时间执行对应的操作、然后tick、保证接上以前的任务
	now := time.Now().Unix()
	if t.StartTaskTime == 0 {
		t.StartTaskTime = now
	}

	//如果任务已经结束、则忽略
	if t.Cycle != -1 && t.StartTaskTime+t.Cycle < now {
		return
	}
	if (t.EndTime != 0 && t.EndTime < now) || (t.StartTime != 0 && t.StartTime < now-t.Duration) || (t.StartTime == 0 && t.EndTime == 0) {
		t.StartTime = now
		t.EndTime = now + t.Duration
	}
	//任务中断过
	if t.EndTime != 0 && (now > t.StartTime && t.EndTime > now && t.StartTime > now-t.Duration && t.EndTime-now < t.Duration) {
		log.Print("delay Tc" + t.TaskStr)
		t.Interrupted = 1
		time.AfterFunc(time.Duration(t.EndTime-now)*time.Second, func() {
			log.Print("delay Tc" + t.TaskStr)
			//到点了执行一次
			do(t)
			tc.AddTc(t.Task)

		})
	} else {
		tc.newTask(t)
	}
}

//开始 任务
func (tc *Timechannel) Start() {
	as = true
	//默认死循环
	for {
		select {
		case <-tc.T.C:
			// tc.check() 数量小还可以、数量很大的话主协程不能这么做、只能在时间到了加
		case d := <-tc.C:
			//增加
			if d.Signal == ADDTASK {
				tc.addTc((*Task)(d.Data))
			}
			//更新
			if d.Signal == UPDATETASK {
				tc.updateTc((*Task)(d.Data))
			}
			//删除
			if d.Signal == DELTASK {
				tc.delTc(*(*string)(d.Data))
			}
		}
	}

}

//延时任务
func (tc *Timechannel) newDelayTak(t *TimeTask) {
	ticker := time.NewTimer(time.Duration(t.Duration) * time.Second)
	c := make(chan Chanl, 1)
	t.TD = ticker
	t.C = c
	go func(t *TimeTask) {
		defer t.TD.Stop()
		for {
			select {
			case <-t.TD.C:
				//任务的task
				do(t)
			case stop := <-t.C:
				if UPDATETASK == stop.Signal {
					return
				}
				//删除操作
				end(t.Task, stop)
				return
			}
		}
	}(t)
}

//定时任务
func (tc *Timechannel) newTask(t *TimeTask) {
	log.Println("task info:", t)
	if t.Duration == 0 {
		t.Duration = DefaultDuration
	}
	//然后将中断恢复
	t.Interrupted = 0
	if t.Delay != 0 {
		tc.newDelayTak(t)
		return
	}

	ticker := time.NewTicker(time.Duration(t.Duration) * time.Second)
	c := make(chan Chanl, 1)
	t.T = ticker
	t.C = c
	go func(t *TimeTask) {
		defer t.T.Stop()
		for {
			select {
			case <-t.T.C:
				//任务的task
				do(t)
			case stop := <-t.C:
				if stop.Signal == UPDATETASK {
					return
				}
				end(t.Task, stop)
				return
			}
		}
	}(t)
}

//实现接口

//更新
func (tc *Timechannel) UpdateTc(task *Task) error {
	if !as {

	}
	tc.C <- Chanl{Signal: UPDATETASK, Data: unsafe.Pointer(task)}
	return nil
}

//更新任务
func (tc *Timechannel) updateTc(task *Task) error {
	for e := tc.tt.Front(); e != nil; {
		v := e.Value.(*TimeTask)
		n := e.Next()
		if v.Tid == task.Tid {
			tc.tt.Remove(e)
			v.C <- Chanl{Signal: TIMEOUTASK}
			autoUpdate(task, v.Task)
			tc.addTc(task)
			return nil
		}
		e = n
	}
	//重开新的定时器
	return errors.New("fail")
}

//添加
func (tc *Timechannel) AddTc(task *Task) error {
	if !as {
		tc.addTc(task)
		return nil
	}
	tc.C <- Chanl{Signal: ADDTASK, Data: unsafe.Pointer(task)}
	return nil
}

//往任务链表里加数据
func (tc *Timechannel) addTc(t *Task) error {
	var tmp TimeTask
	if t.Tid == "" {
		guid := xid.New().String()
		t.Tid = guid
	}
	tmp.Task = t
	tc.CheckAndTask(&tmp)
	//加入全局
	tc.tt.PushBack(&tmp)
	return nil
}

//删除任务
func (tc *Timechannel) DelTc(tid string) error {
	tc.C <- Chanl{Signal: DELTASK, Data: unsafe.Pointer(&tid)}
	return nil
}

//
//删除任务
func (tc *Timechannel) delTc(tid string) error {
	for e := tc.tt.Front(); e != nil; {
		v := e.Value.(*TimeTask)
		n := e.Next()
		if v.Tid == tid {
			tc.tt.Remove(e)
			v.C <- Chanl{Signal: DELTASK}
			return nil
		}
		e = n
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
