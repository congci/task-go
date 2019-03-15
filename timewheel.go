//时间轮 实现的超时
//原理 轮询时间轮、每秒走一个槽、然后遍历槽上链表看是否过期
//重点是判断任务放在哪个槽上

package task

import (
	"container/list"
	"errors"
	"log"
	"sync"
	"time"
	"unsafe"

	"github.com/rs/xid"
)

type Timewheel struct {
	solts        []*list.List
	slotsNum     int
	T            *time.Ticker   //定时器
	C            chan Chanl     //定时器传值
	currentTick  int            //现在的时针
	tickduration int            //间隔
	taskmap      map[string]int //task的id映射在槽的index
	taskqueue    taskqueue      //任务池
}

var rww sync.RWMutex

func (tw *Timewheel) CheckAndAddTask(t *TimeTask) {
	now := time.Now().Unix()
	//如果任务已经结束、则忽略
	if t.StartTaskTime == 0 {
		t.StartTaskTime = now
	}

	if t.Cycle != -1 && t.StartTaskTime+t.Cycle < now {
		return
	}
	//代表这个任务记录有问题、1\结束时间小于现在 2、开始时间小于 最近一个周期
	if (t.EndTime != 0 && t.EndTime < now) || (t.StartTime != 0 && t.StartTime < now-t.Duration) || (t.StartTime == 0 && t.EndTime == 0) {
		t.StartTime = now
		t.EndTime = now + t.Duration
	}
	//任务中断过
	if t.EndTime != 0 && (now > t.StartTime && t.EndTime > now && t.StartTime > now-t.Duration && t.EndTime-now < t.Duration) {
		log.Print("delay Tc" + t.TaskStr)
		//走信号
		time.AfterFunc(time.Duration(t.EndTime-now)*time.Second, func() {
			do(t)
			//加入队伍
			tw.AddTc(t.Task)
		})
	} else {
		tw.newTask(t)
	}
}

//如果有任务进来、则执行
func (tw *Timewheel) Execs() {
	for {
		select {
		case data := <-tw.taskqueue.queue:
			t := (*TimeTask)(data)
			do(t)
		}
	}
}
func (tw *Timewheel) Start() {
	as = true
	for {
		select {
		case <-tw.T.C:
			//每秒执行一次循环
			tw.Exec()
		case d := <-tw.C:
			tw.action(d)
		}
	}
}

//如果有值
func (tw *Timewheel) action(d Chanl) {
	//增加
	if d.Signal == ADDTASK {
		tw.addTc((*Task)(d.Data))
	}
	//更新
	if d.Signal == UPDATETASK {
		tw.updateTc((*Task)(d.Data))
	}
	//删除
	if d.Signal == DELTASK {
		tw.delTc(*(*string)(d.Data))
	}
}

func newTimeWheel() *Timewheel {
	t := new(Timewheel)
	t.initT()
	return t
}

//list表
func (tw *Timewheel) initT() {
	tw.slotsNum = 3600
	tw.solts = make([]*list.List, tw.slotsNum)
	for i := 0; i < tw.slotsNum; i++ {
		tw.solts[i] = list.New()
	}
	tw.taskqueue.num = 20

	//代表启动任务池
	if tw.taskqueue.num != 0 {
		tw.taskqueue.queue = make(chan unsafe.Pointer, 1000)
		for i := 0; i < tw.taskqueue.num; i++ {
			go tw.Execs()
		}
	}
	tw.currentTick = 0
	tw.tickduration = 1
	tw.taskmap = make(map[string]int)
	c := make(chan Chanl, 1)
	tw.T = time.NewTicker(1 * time.Second)
	tw.C = c
}

//循环处理
//检查每个任务的cycle_num 如果是 0 则 执行、否则 -1
func (tw *Timewheel) Exec() {
	ll := tw.solts[tw.currentTick]
	if ll == nil {
		return
	}
	for e := ll.Front(); e != nil; {
		v := e.Value.(*TimeTask)
		if v.cyclenum != 0 {
			v.cyclenum--
			e = e.Next()
		} else {
			//如果是0、则执行、并且删除、如果不是delay而是tick则再次创建
			n := e.Next()
			ll.Remove(e)
			//如果有协程池、则走任务执行池
			if tw.taskqueue.num != 0 && cap(tw.taskqueue.queue)-len(tw.taskqueue.queue) > 0 {
				//因为都是主协操作、因此不用考虑其他情况
				tw.taskqueue.queue <- unsafe.Pointer(v)
			} else {
				//如果不走协程池或者队列满了、则新开新的现场
				go do(v)
			}
			// 不是延时 并且没有过期
			if _, ok := tw.taskmap[v.Tid]; ok {
				delete(tw.taskmap, v.Tid)
			}
			if (v.Task.Delay == 0 && v.StartTaskTime+v.Cycle > time.Now().Unix()) || v.Cycle == -1 {
				tw.addTc(v.Task)
			} else {
				//过期
				v.EndFunc(v.Task, Chanl{})
			}
			e = n
		}
	}
	if tw.currentTick < tw.slotsNum-1 {
		tw.currentTick++
	}
	if tw.currentTick == tw.slotsNum-1 {
		tw.currentTick = 0
	}
}

//实现接口
//添加
func (tw *Timewheel) AddTc(t *Task) error {
	if !as {
		tw.addTc(t)
		return nil
	}
	tw.C <- Chanl{Signal: ADDTASK, Data: unsafe.Pointer(t)}
	return nil
}
func (tw *Timewheel) newTask(t *TimeTask) {
	t.Interrupted = 0
	d := t.Duration //按照秒
	t.cyclenum = int(d) / tw.slotsNum
	pos := (tw.currentTick + int(t.Duration)/tw.tickduration) % tw.slotsNum
	//如果任务id为0 则生成一个
	if t.Tid == "" {
		guid := xid.New().String()
		t.Tid = guid
	}

	tw.taskmap[t.Tid] = pos
	tw.solts[pos].PushBack(t)
}

//添加
func (tw *Timewheel) addTc(task *Task) error {
	var tmp TimeTask
	tmp.Task = task
	tw.CheckAndAddTask(&tmp)
	return nil
}

//删除
func (tw *Timewheel) DelTc(id string) error {
	if !as {
		return errors.New("")
	}
	tw.C <- Chanl{Signal: DELTASK, Data: unsafe.Pointer(&id)}
	return nil
}

//删除
func (tw *Timewheel) delTc(tid string) error {
	if index, ok := tw.taskmap[tid]; ok {
		delete(tw.taskmap, tid)
		for e := tw.solts[index].Front(); e != nil; {
			v := e.Value.(*TimeTask)
			n := e.Next()
			if v.Task.Tid == tid {
				tw.solts[index].Remove(e)
				//如果有附加的小任务、则也直接删除
				if v.ExTendTids != nil {
					//循环删除
					for _, etid := range v.ExTendTids {
						tw.delTc(etid)
					}
				}
				break
			}
			e = n
		}
	}
	return nil
}

//更新
func (tw *Timewheel) UpdateTc(task *Task) error {
	if !as {
		return errors.New("")
	}
	tw.C <- Chanl{Signal: UPDATETASK, Data: unsafe.Pointer(task)}
	return nil
}

//更新
func (tw *Timewheel) updateTc(task *Task) error {
	tid := task.Tid
	if index, ok := tw.taskmap[tid]; ok {
		for e := tw.solts[index].Front(); e != nil; {
			v := e.Value.(*TimeTask)
			//直接删除、然后新建
			n := e.Next()
			if v.Task.Tid == tid {
				autoUpdate(task, v.Task)
				tw.solts[index].Remove(e)
				tw.addTc(task)
				break
			}
			e = n

		}
	}
	return nil
}

//获取全部task、包括开始中断中的
func (tw *Timewheel) GetAllTasks() []*Task {
	var tmp []*Task
	for _, v := range tw.solts {
		for e := v.Front(); e != nil; e = e.Next() {
			val := e.Value.(*TimeTask)
			tmp = append(tmp, val.Task)
		}
	}
	return tmp
}
