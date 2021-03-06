package task

import (
	"container/list"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unsafe"

	"github.com/rs/xid"
)

//time + channel - 底层是最小堆
type Timechannel struct {
	tt *list.List //任务保存的地方
	C  chan Chanl //主线程通知信号
}

func newTimeChannel(p *Param) *Timechannel {
	t := new(Timechannel)
	t.tt = list.New()
	t.C = make(chan Chanl, 1)
	return t
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

	//第一次添加有延时
	if t.Delay != 0 {
		t.Delay = 0
		//timeafter是另一个协程、因此只能到主操作
		time.AfterFunc(time.Duration(t.Delay), func() {
			log.Print("delay Tc" + t.TaskStr)
			tc.C <- Chanl{Signal: DELAYTASK, Data: unsafe.Pointer(t)}
		})
		return
	}

	//如果初次加载的任务中断过
	if t.EndTime != 0 && now > t.StartTime && t.EndTime > now && t.StartTime > now-t.Duration && t.EndTime-now < t.Duration {
		t.Delay = 0
		time.AfterFunc(time.Duration(t.EndTime-now)*time.Second, func() {
			log.Print("delay Tc" + t.TaskStr)
			do(t)
			tc.C <- Chanl{Signal: DELAYTASK, Data: unsafe.Pointer(t)}
		})
		return
	}
	tc.newTask(t)
}

//默认死循环、添加删除都是在这个协程里、不能在其他协程里
func (tc *Timechannel) Start() {
	//监听信号
	go tc.signal()
	as = true
	for {
		select {
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
				tc.delTc(*(*string)(d.Data), true)
			}

			//专门处理开始延时的任务
			if d.Signal == DELAYTASK {
				tc.delayTc((*TimeTask)(d.Data))
			}
		}
	}

}

//定时任务
func (tc *Timechannel) newTask(t *TimeTask) {
	log.Println("task info:", t)

	ticker := time.NewTicker(time.Duration(t.Duration) * time.Second)
	c := make(chan Chanl, 1)
	t.T = ticker
	t.C = c
	go func(t *TimeTask) {
		defer t.T.Stop()
		for {
			select {
			case <-t.T.C:
				if ((t.StartTaskTime+t.Cycle > time.Now().Unix() && t.Cycle != -1) || t.Cycle == -1) && (t.LimitNum == 0 || (t.LimitNum != 0 && t.LimitNum > 1 && t.Num < t.LimitNum)) {
					//任务的task
					do(t)
				} else {
					if t.del {
						return
					}
					//过期直接退出
					tc.delTc(t.Tid, false) //先删除任务
					end(t.Task, Chanl{Signal: TIMEOUTASK})
				}
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
		return errors.New("task no start")
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
			v.C <- Chanl{Signal: UPDATETASK}
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
	if task.Duration == 0 || task.Cycle == 0 {
		return errors.New("add task fail no du no cy")
	}
	if !as {
		tc.addTc(task)
		return nil
	}
	tc.C <- Chanl{Signal: ADDTASK, Data: unsafe.Pointer(task)}
	return nil
}

//专门添加延时的
func (tc *Timechannel) delayTc(t *TimeTask) {
	//延时执行的如果已经删除、则直接不加
	if t.del {
		return
	}
	tc.newTask(t)
}

//往任务链表里加数据
func (tc *Timechannel) addTc(t *Task) error {
	var tmp TimeTask
	if t.Tid == "" {
		guid := xid.New().String()
		t.Tid = guid
	}
	tmp.Task = t
	//加入全局
	tc.tt.PushBack(&tmp)
	tc.CheckAndTask(&tmp)
	return nil
}

//删除任务
func (tc *Timechannel) DelTc(tid string) error {
	tc.C <- Chanl{Signal: DELTASK, Data: unsafe.Pointer(&tid)}
	return nil
}

//
//删除任务\mode为false仅仅指删除
func (tc *Timechannel) delTc(tid string, mode bool) error {
	for e := tc.tt.Front(); e != nil; {
		v := e.Value.(*TimeTask)
		n := e.Next()
		if v.Tid == tid {
			tc.tt.Remove(e)
			v.del = true //利用指针的特性、为延后加入期间删除的加个开关、如果在timeafter期间删除、则不再加入
			//延期加入的此时没有c
			if mode && v.C != nil {
				v.C <- Chanl{Signal: DELTASK}
			}
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

//信号机制、主要是不断开任务、在结束的时候把任务刷到存储里
func (tc *Timechannel) signal() {
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGUSR2)
	for {
		<-sig
		tc.StoreDFunc()
		//暂时断开
		os.Exit(1)
	}
}

//默认存储
func (tc *Timechannel) StoreDFunc() {
	for e := tc.tt.Front(); e != nil; e = e.Next() {
		val := e.Value.(*TimeTask)
		if val.StoreFunc != nil && !val.IsExtendTask {
			val.StoreFunc(val.Task)
		}
	}
}
