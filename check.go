package task

import (
	"log"
	"time"
)

//做一个检测程序、备份的、主要停止应该是每次任务执行时、主动判断、
func check() {
	//如果超时则直接停止
	for {
		RW.RLock()
		now := time.Now().Unix()
		for i := 0; i < len(Tc); i++ {
			//(Tc[i].Duration < 100 && Tc[i].Num > 3) 为特殊设定、测试用的、可以求掉
			if (Tc[i].Cycle != -1 && Tc[i].StartTaskTime+Tc[i].Cycle < now) || (Tc[i].LimitNum != 0 && Tc[i].Num > Tc[i].LimitNum) || (Tc[i].Duration < 100 && Tc[i].Num > 3) {
				log.Print("close Tc" + Tc[i].TaskStr)
				Tc[i].C <- TIMEOUTASK
				if i+1 > len(Tc) {
					Tc = Tc[:i]
				} else {
					Tc = append(Tc[:i], Tc[i+1:]...)
				}
			}
		}
		RW.RUnlock()
		time.Sleep(1 * time.Second)
	}
}
