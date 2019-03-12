//时间轮 实现的超时
package timewheel

import (
	"container/list"
	"sync"
	"time"
)

const ticks = 3600   //一圈的长度
var tickDuration = 1 //一圈的间隔

//3600s solt
//每个solt是个切片的

var solts [ticks]*list.List

type TicketW struct {
	//一个
}

type TWheel struct {
	T *time.Ticker     //定时器
	C chan interface{} //定时器传值
}

var currentTick int
var RWW sync.RWMutex

var taskIndexMap map[int]int //任务和index的对应关系

func Start() {
	initT()

}

//循环、每秒执行一次
func wheel() {
}

//list表
func initT() *TWheel {
	for k, _ := range solts {
		solts[k] = list.New()
	}

	c := make(chan interface{}, 1)
	return &TWheel{
		T: time.NewTicker(1 * time.Second),
		C: c,
	}
}

//循环处理
//检查每个任务的cycle_num 如果是 0 则 执行、否则 -1
func Exec() {

}
