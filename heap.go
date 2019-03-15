//最小堆
//原理 将tick或者timeout以时间注册、然后取出最小堆的最小时间、然后在时间内sleep、到期了苏醒执行取出的任务、如果是定时、则再次注册、依次循环

package task

import (
	"container/list"
	"time"
)

type Timeheap struct {
}

const mintime = 1 * time.Microsecond

//时间 + 链表任务
var tts map[int64]*list.List

func Start() {
	for {

	}
}
