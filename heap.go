//最小堆
//原理 将tick或者timeout以时间注册、然后取出最小堆的最小时间、然后在时间内sleep、到期了苏醒执行取出的任务、如果是定时、则再次注册、依次循环

package task

import (
	"container/list"
	"errors"
	"sync"
	"time"
	"unsafe"
)

type Timeheap struct {
	C         chan Chanl
	taskqueue taskqueue    //任务池
	mintime   int64        //每次循环处理的最小事件
	rw        sync.RWMutex // 此处就需要处理
}

//添加的时候判断是否比这个小、如果比这个小、则最小变为这个、时间差
const mintime = 20 * time.Microsecond

//执行时间点 + 链表任务
var tts map[int64]*list.List

func (th *Timeheap) Start() {
	//获取数据
	for {
		//获取最小的pop时间、然后全部过期、然后执行过期的任务、然后sleep
		th.rw.Lock()

		time.Sleep(time.Duration(th.mintime) * time.Microsecond)
	}

	go func() {
		for {
			select {
			//执行动作
			case d := <-th.C:
				//增加
				if d.Signal == ADDTASK {
					th.addTc((*Task)(d.Data))
				}
				//更新
				if d.Signal == UPDATETASK {
					th.updateTc((*Task)(d.Data))
				}
				//删除
				if d.Signal == DELTASK {
					th.delTc(*(*string)(d.Data))
				}
			}
		}
	}()
}

//添加接口

func (th *Timeheap) AddTc(t *Task) error {
	if !as {
		th.addTc(t)
		return nil
	}
	th.C <- Chanl{Signal: ADDTASK, Data: unsafe.Pointer(t)}
	return nil
}

func (th *Timeheap) UpdateTc(t *Task) error {
	if !as {
		return errors.New("")
	}
	th.C <- Chanl{Signal: UPDATETASK, Data: unsafe.Pointer(t)}
	return nil
}
func (th *Timeheap) DelTc(tid string) error {
	if !as {
		return errors.New("")
	}
	th.C <- Chanl{Signal: DELTASK, Data: unsafe.Pointer(&tid)}
	return nil
}

//实现逻辑、暂时只实现秒
func (th *Timeheap) addTc(t *Task) {
	if (t.EndTime-time.Now().Unix())*1e6 < int64(mintime) {
		th.mintime = int64(mintime)
	} else {
		th.mintime = (t.EndTime - time.Now().Unix()) * 1e6
	}
}
func (th *Timeheap) updateTc(t *Task) {}
func (th *Timeheap) delTc(tid string) {}
