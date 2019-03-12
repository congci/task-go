//时间轮 实现的超时
package task

import (
	"container/list"
	"log"
	"sync"
	"time"
	"unsafe"
)

type Timewheel struct {
	tt           *list.List //主要是初始化的时候的任务
	solts        []*list.List
	slotsNum     int
	T            *time.Ticker     //定时器
	C            chan interface{} //定时器传值
	currentTick  int              //现在的时针
	tickduration time.Duration    //间隔
	taskmap      map[int]int      //task的id映射在槽的index
}

type TimeWTask struct {
	cyclenum int //第几圈
	index    int //索引
	*Task
}

var rww sync.RWMutex

var taskIndexMap map[int]int //任务和index的对应关系

func (tw *Timewheel) LoadTasks() {
	if tw.tt.Len() == 0 {
		return
	}
	//结束时间 - 当前时间 定时器执行、如果到了时间执行对应的操作、然后tick、保证接上以前的任务
	now := time.Now().Unix()
	for e := tw.tt.Front(); e != nil; e = e.Next() {
		v := e.Value.(*Task)
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

	for e := tw.tt.Front(); e != nil; e = e.Next() {
		v := e.Value.(*Task)
		if v.Interrupted == 1 {
			time.AfterFunc(time.Duration(v.EndTime-now)*time.Second, func() {
				tw.do(v)
				//加入队伍
				tw.addTc(v)
			})
		} else {
			tw.addTc(v)
		}
	}

}

func (tw *Timewheel) Start() {
	for {
		select {
		case <-tw.T.C:
			//每秒执行一次循环
			tw.Exec()
		case data := <-tw.C:
			d := data.(chanl)
			//增加
			if d.signal == ADDTASK {
				tw.addTc((*Task)(d.data))
			}
			//更新
			if d.signal == UPDATETASK {
				tw.updateTc((*Task)(d.data))
			}
			//删除
			if d.signal == DELTASK {
				tw.delTc(*(*int)(d.data))
			}
		}
	}

}

func newTimeWheel() *Timewheel {
	t := new(Timewheel)
	t.initT()
	return t
}

//list表
func (tw *Timewheel) initT() {
	for k, _ := range tw.solts {
		tw.solts[k] = list.New()
	}

	tw.taskmap = make(map[int]int)
	c := make(chan interface{}, 1)
	tw.T = time.NewTicker(1 * time.Second)
	tw.C = c
}

//循环处理
//检查每个任务的cycle_num 如果是 0 则 执行、否则 -1
func (tw *Timewheel) Exec() {
	if tw.currentTick < tw.slotsNum {
		tw.currentTick++
	}
	if tw.currentTick == tw.slotsNum-1 {
		tw.currentTick = 0
	}
	ll := tw.solts[tw.currentTick]
	if ll == nil {
		return
	}
	for e := ll.Front(); e != nil; {
		v := e.Value.(*TimeWTask)
		if v.cyclenum != 0 {
			v.cyclenum--
		} else {
			//如果是0、则执行、并且删除、如果不是delay而是tick则再次创建
			ll.Remove(e)
			if v.Task.Delay == 0 {
				tw.AddTc(v.Task)
			}
			go tw.do(v.Task)
		}
		e = e.Next()
	}
}

func (tw *Timewheel) do(task *Task) {

}

//添加
func (tw *Timewheel) AddTc(task *Task) error {
	tw.C <- chanl{signal: ADDTASK, data: unsafe.Pointer(task)}
	return nil
}

//添加
func (tw *Timewheel) addTc(task *Task) error {
	var tmp TimeWTask
	d := task.Duration //按照秒
	tmp.cyclenum = int(d) / tw.slotsNum
	pos := (tw.currentTick + int(task.Duration)/int(tw.tickduration.Seconds())) % tw.slotsNum
	tmp.index = pos
	tw.solts[pos].PushBack(&tmp)
	return nil
}

func (tw *Timewheel) AddOnlyTask(task *Task) {
	tw.tt.PushBack(task)
}

//删除
func (tw *Timewheel) DelTc(id int) error {
	tw.C <- chanl{signal: DELTASK, data: unsafe.Pointer(&id)}
	return nil
}

//删除
func (tw *Timewheel) delTc(id int) error {
	if index, ok := tw.taskmap[id]; ok {
		for e := tw.solts[index].Front(); e != nil; {
			v := e.Value.(*TimeWTask)
			if v.Task.Id == id {
				tw.solts[index].Remove(e)
			}
			break
		}
	}
	return nil
}

//更新
func (tw *Timewheel) UpdateTc(task *Task) error {
	tw.C <- chanl{signal: DELTASK, data: unsafe.Pointer(task)}
	return nil
}

//更新
func (tw *Timewheel) updateTc(task *Task) error {
	id := task.Id
	if index, ok := tw.taskmap[id]; ok {
		for e := tw.solts[index].Front(); e != nil; {
			v := e.Value.(*TimeWTask)
			//直接删除、然后新建
			if v.Task.Id == id {
				tw.solts[index].Remove(e)
			}
			tw.AddTc(task)
			break
		}
	}
	return nil
}

//获取全部task、包括开始中断中的
func (tc *Timewheel) GetAllTasks() []*Task {
	var tmp []*Task
	for _, v := range tc.solts {
		for e := v.Front(); e != nil; e = e.Next() {
			val := e.Value.(*TimeWTask)
			tmp = append(tmp, val.Task)
		}
	}
	return tmp
}
