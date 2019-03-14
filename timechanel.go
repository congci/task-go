package task

import (
	"container/list"
	"errors"
	"log"
	"time"
	"unsafe"
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
}

//如果有有开始任务、那么执行
func (tc *Timechannel) prestart() {
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
		if v.EndTime != 0 && (now > v.StartTime && v.EndTime > now && v.EndTime-now < v.Duration) {
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
				do(v)
			})
		} else {
			tc.newTask(v)
		}
	}
}

//开始 任务
func (tc *Timechannel) Start() {
	tc.prestart()
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
				tc.delTc(*(*int)(d.Data))
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
				if DELTASK == stop.Signal {
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
	if t.Delay != 0 {
		tc.newDelayTak(t)
		return
	}

	//然后将中断恢复
	t.Interrupted = 0
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
				for e := tc.tt.Front(); e != nil; {
					v := e.Value.(*TimeTask)
					if v.Id == t.Id {
						tc.tt.Remove(e)
					}
					e = e.Next()
				}
				end(t.Task, stop)
				return
			}
		}
	}(t)
}

//每秒扫描一次、检查数据是否过期
func (tc *Timechannel) check() {
	now := time.Now().Unix()
	for e := tc.tt.Front(); e != nil; {
		t := e.Value.(*TimeTask)
		if (t.Cycle != -1 && t.StartTaskTime+t.Cycle < now) || (t.LimitNum != 0 && t.Num > t.LimitNum) {
			//发送数据
			t.C <- Chanl{Signal: TIMEOUTASK}
			tc.tt.Remove(e)
		}
		e = e.Next()
	}
}

//实现接口

//只在加载前操作、因此没有冲突
func (tc *Timechannel) AddOnlyTask(t *Task) {
	var tmp TimeTask
	formatTask(t)

	tmp.Task = t
	tc.tt.PushBack(&tmp)
}

//更新
func (tc *Timechannel) UpdateTc(task *Task) error {
	tc.C <- Chanl{Signal: DELTASK, Data: unsafe.Pointer(task)}
	return nil
}

//更新任务
func (tc *Timechannel) updateTc(task *Task) error {
	formatTask(task)

	for e := tc.tt.Front(); e != nil; {
		v := e.Value.(*TimeTask)
		if v.Id == task.Id {
			tc.tt.Remove(e)
			v.C <- Chanl{Signal: TIMEOUTASK}
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
	tc.C <- Chanl{Signal: ADDTASK, Data: unsafe.Pointer(task)}
	return nil
}

//往任务链表里加数据
func (tc *Timechannel) addTc(t *Task) error {
	var tmp TimeTask
	formatTask(t)
	tmp.Task = t
	tc.tt.PushBack(&tmp)
	tc.newTask(&tmp)
	return nil
}

//删除任务
func (tc *Timechannel) DelTc(id int) error {
	tc.C <- Chanl{Signal: DELTASK, Data: unsafe.Pointer(&id)}
	return nil
}

//
//删除任务
func (tc *Timechannel) delTc(id int) error {
	for e := tc.tt.Front(); e != nil; {
		v := e.Value.(*TimeTask)
		if v.Id == id {
			tc.tt.Remove(e)
			v.C <- Chanl{Signal: TIMEOUTASK}
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
