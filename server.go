package kernel

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Res struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

var failRes = Res{Code: 100, Msg: "fail"}
var successRes = Res{Code: 0, Msg: "success"}

func success(w *http.ResponseWriter) {
	res, _ := json.Marshal(&successRes)
	(*w).Write(res)
	return
}

func fail(w *http.ResponseWriter) {
	res, _ := json.Marshal(&failRes)
	(*w).Write(res)
	return
}

func Server(port string) {
	http.HandleFunc("/pushsub", addTc)      // 增加任务
	http.HandleFunc("/updatesub", updateTc) //更新任务
	http.HandleFunc("/status", status)      //获取状态
	http.HandleFunc("/delsub", delTc)       //删除任务
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

//添加任务、添加任务和
func addTc(w http.ResponseWriter, req *http.Request) {
	b, _ := ioutil.ReadAll(req.Body)
	defer req.Body.Close()

	var task TimeTask
	err := json.Unmarshal(b, &task)
	if err != nil {
		fail(&w)
		return
	}

	for _, v := range Tc {
		if v.Id == task.Id {
			return
		}
	}

	task.TaskStr = string(b)
	if task.Create_time != "" {
		s, _ := time.Parse("2006-01-02 15:04:05", task.Create_time)
		//任务开始时间和任务周期必须有
		task.StartTaskTime = s.Unix()
	} else {
		task.StartTaskTime = time.Now().Unix()
	}
	//开始任务、这里忽略线程安全、
	RW.Lock()
	newTask(&task)
	RW.Unlock()
	success(&w)
}

//修改任务
func updateTc(w http.ResponseWriter, req *http.Request) {
	b, _ := ioutil.ReadAll(req.Body)
	defer req.Body.Close()

	var task TimeTask
	err := json.Unmarshal(b, &task)
	if err != nil {
		fail(&w)
		return
	}

	task.TaskStr = string(b)
	if task.Create_time != "" {
		s, _ := time.Parse("2006-01-02 15:04:05", task.Create_time)
		//任务开始时间和任务周期必须有
		task.StartTaskTime = s.Unix()
	} else {
		task.StartTaskTime = time.Now().Unix()
	}

	RW.Lock()
	defer RW.Unlock()
	for k, v := range Tc {
		if v.Id == task.Id {
			// 废弃这个协程任务并且开一个新的定时器
			v.C <- RENEWTASK
			if k+1 > len(Tc) {
				Tc = append(Tc[:k], Tc[k+1:]...)
			} else {
				Tc = Tc[:k]
			}
			newTask(&task)
			success(&w)
			return
		}
	}

	//如果是中断延时的任务则直接更改
	for k, v := range delayTc {
		if v.Id == task.Id {
			delayTc[k] = &task
		}
	}

	fail(&w)
}

//删除任务
func delTc(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	if len(req.Form["id"]) < 0 {
		var r = Res{Code: 100, Msg: "fail"}
		res, _ := json.Marshal(&r)
		w.Write(res)
	}
	id, _ := strconv.Atoi(req.Form["id"][0])
	RW.Lock()
	defer RW.Unlock()

	for k, v := range Tc {
		if v.Id == id {
			// 废弃这个协程任务
			v.C <- DELTASK
			if k+1 > len(Tc) {
				Tc = append(Tc[:k], Tc[k+1:]...)
			} else {
				Tc = Tc[:k]
			}
			success(&w)
			return
		}
	}
	fail(&w)
}

//获取状态
func status(w http.ResponseWriter, req *http.Request) {
	b, _ := json.Marshal(Tc)
	w.Write(b)
}
