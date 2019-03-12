package task

import (
	"log"
	"time"
)

var retrytt = NewQueue()

// -- 异常处理
//func 重试 //队列获取然后执行
func exception() {
	for {
		v := retrytt.pop()
		if v == nil {
			time.Sleep(1 * time.Second)
			continue
		}
		t := v.(*TimeTask)
		//数据
		if t.RetryNum == 3 {
			log.Print("retry num 3 ", v)
			continue
		}
		t.RetryNum++
	}

}
