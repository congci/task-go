package task

import "time"

//做一个检测程序、备份的、主要停止应该是每次任务执行时、主动判断、
func check() {
	for {
		now := time.Now().Unix()
		for e := tc.data.Front(); e != nil; {
			t := e.Value.(*TimeTask)
			if (t.Cycle != -1 && t.StartTaskTime+t.Cycle < now) || (t.LimitNum != 0 && t.Num > t.LimitNum) {
				tc.data.Remove(e)
			}
			e = e.Next()
		}
		time.Sleep(1 * time.Second)
	}
}
