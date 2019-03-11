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

func (m Access) Pop() *TimeTask {
	l := len(queue)
	if l == 0 {
		return nil
	}
	task := queue[l]
	queue = queue[0:(l - 1)]
	return task
}

func (m Access) Push(v interface{}) error {
	task := v.(*TimeTask)
	queue = append(queue, task)
	return nil
}

// -- 异常处理
//func 重试 //队列获取然后执行
func exception() {
	var m Access
	for {
		v := m.Pop()
		if v == nil {
			time.Sleep(1 * time.Second)
			continue
		}
		//数据
		if v.RetryNum == 3 {
			log.Print("retry num 3 ", v)
			continue
		}
		v.RetryNum++
		Do(v)
	}

}
