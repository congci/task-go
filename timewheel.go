//时间轮 实现的超时
//原理 轮询时间轮、每秒走一个槽、然后遍历槽上链表看是否过期
//重点是判断任务放在哪个槽上

package task

import (
	"container/list"
	"errors"
	"log"
	"time"
	"unsafe"

	"github.com/rs/xid"
)

//默认槽数量
const DEFAULTSOLTSNUM = 3600
const QueueCap = 1000

type Timewheel struct {
	solts        []*list.List
	slotsNum     int
	T            *time.Ticker   //定时器
	C            chan Chanl     //定时器传值
	currentTick  int            //现在的时针
	tickduration time.Duration  //间隔
	taskmap      map[string]int //task的id映射在槽的index
	taskqueue    taskqueue      //任务池
}

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
	//任务中断过\延迟启动、加入到数据里
	if t.EndTime != 0 && now > t.StartTime && t.EndTime > now && t.StartTime > now-t.Duration && t.EndTime-now < t.Duration {
		log.Print("delay Tc" + t.TaskStr)
		t.Delay = t.EndTime - now
	}
	tw.newTask(t)

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
			//时间轮指针往前走
			tw.Exec()

			//执行动作
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

//默认不走协程池
func newTimeWheel(p *Param) *Timewheel {
	//初始
	if p == nil {
		p = &Param{
			SlotsNum:     DEFAULTSOLTSNUM,
			QueueNum:     0,
			QueueCap:     QueueCap,
			Tickduration: 1 * time.Second, //默认一秒都一次
		}
	}

	if p.SlotsNum == 0 {
		p.SlotsNum = DEFAULTSOLTSNUM
	}

	if p.QueueCap == 0 {
		p.QueueCap = QueueCap
	}

	if p.Tickduration == 0 {
		p.Tickduration = 1 * time.Second
	}

	t := new(Timewheel)
	t.slotsNum = p.SlotsNum
	t.solts = make([]*list.List, t.slotsNum)
	for i := 0; i < t.slotsNum; i++ {
		t.solts[i] = list.New()
	}

	t.taskqueue.num = p.QueueNum

	//代表启动任务池
	if t.taskqueue.num != 0 {
		t.taskqueue.queue = make(chan unsafe.Pointer, p.QueueCap)
		for i := 0; i < t.taskqueue.num; i++ {
			go t.Execs()
		}
	}

	t.currentTick = 0

	t.tickduration = p.Tickduration

	t.taskmap = make(map[string]int)
	c := make(chan Chanl, 1)
	t.T = time.NewTicker(t.tickduration)
	t.C = c
	return t
}

//循环处理
//检查每个任务的cycle_num 如果是 0 则 执行、否则 -1
func (tw *Timewheel) Exec() {
	ll := tw.solts[tw.currentTick]
	if tw.currentTick < tw.slotsNum-1 {
		tw.currentTick++
	}
	if tw.currentTick == tw.slotsNum-1 {
		tw.currentTick = 0
	}
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
			//将偏移置为0、再次加入的时候就不用偏移了
			v.Delay = 0
			//如果有协程池、则走任务执行池
			if tw.taskqueue.num != 0 && cap(tw.taskqueue.queue)-len(tw.taskqueue.queue) > 0 {
				//因为都是主协操作、因此不用考虑其他情况
				tw.taskqueue.queue <- unsafe.Pointer(v)
			} else {
				//如果不走协程池或者队列满了、则新开新的现场
				go do(v)
			}

			//重新计算入槽
			if _, ok := tw.taskmap[v.Tid]; ok {
				delete(tw.taskmap, v.Tid)
			}
			//只有在不是一次函数的时候并且没过期才会再次加入
			if ((v.StartTaskTime+v.Cycle > time.Now().Unix() && v.Cycle != -1) || v.Cycle == -1) && (v.LimitNum == 0 || (v.LimitNum != 0 && v.LimitNum > 1 && v.Num < v.LimitNum)) {
				tw.addTc(v.Task)
			} else {
				//删除任务
				tw.delTc(v.Tid)
				//过期
				if tw.taskqueue.num != 0 && cap(tw.taskqueue.queue)-len(tw.taskqueue.queue) > 0 {
					//因为都是主协操作、因此不用考虑其他情况
					tw.taskqueue.queue <- unsafe.Pointer(v)
				} else {
					//如果不走协程池或者队列满了、则新开新的现场
					go end(v.Task, Chanl{Signal: TIMEOUTASK})
				}

			}
			e = n
		}
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
	d := t.Duration + t.Delay //按照秒
	t.cyclenum = int(d) / tw.slotsNum
	pos := (tw.currentTick + int(d)/int(tw.tickduration.Seconds())) % tw.slotsNum
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
	if task.Duration == 0 || task.Cycle == 0 {
		return errors.New("add task fail no du no cy")
	}
	var tmp TimeTask
	tmp.Task = task
	tw.CheckAndAddTask(&tmp)
	return nil
}

//删除
func (tw *Timewheel) DelTc(id string) error {
	if !as {
		return errors.New("no runing")
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
				delete(tw.taskmap, v.Tid)
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
		return errors.New("no runing")
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
				tw.delTc(tid) //主任务删除、那么附加任务也删除
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
