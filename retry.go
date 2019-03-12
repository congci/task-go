package task

import (
	"log"
	"time"
)

//异常数据获取 接口、可以自己实现\默认从内存中取
var ExceptionData interface {
	Pop() *TimeTask
	Push(v interface{}) error
}

type Access struct{}

// -- 异常处理
//func 重试 //队列获取然后执行
func exception() {
	for {
		v := queue.Pop()
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
		Do(t)
	}

}
