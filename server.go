package task

import (
	"encoding/json"
	"errors"
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
	task.TaskStr = string(b)
	//初始化task
	formatTask(&task)
	AddTc(&task)
	success(&w)
}

//外部动态添加任务
func AddTc(task *TimeTask) error {
	tc.Push(task)
	newTask(task)
	return nil
}

//更新任务
func UpdateTc(task *TimeTask) error {
	for e := tc.data.Front(); e != nil; {
		v := e.Value.(*TimeTask)
		if v.Id == task.Id {
			tc.data.Remove(e)
			AddTc(task)
			return nil
		}
		e = e.Next()
	}
	//重开新的定时器
	return errors.New("fail")
}

func formatTask(task *TimeTask) {
	now := time.Now().Unix()

	if task.Create_time != "" {
		s, _ := time.Parse("2006-01-02 15:04:05", task.Create_time)
		//任务开始时间和任务周期必须有
		task.StartTaskTime = s.Unix()
	} else {
		task.StartTaskTime = now
	}

	if task.StartTime == 0 {
		task.StartTime = now
	}
	if task.EndTime == 0 {
		task.EndTime = now + task.Duration
	}
	//代表这个任务记录有问题、1\结束时间小于现在 2、开始时间小于 最近一个周期
	if (task.EndTime != 0 && task.EndTime < now) || (task.StartTime != 0 && task.StartTime < now-task.Duration) {
		task.StartTime = now
		task.EndTime = now + task.Duration
	}
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
	formatTask(&task)

	err = UpdateTc(&task)
	if err != nil {
		fail(&w)
	}
	success(&w)
}

//删除任务
func DelTc(id int) error {
	RW.Lock()
	defer RW.Unlock()
	for e := tc.data.Front(); e != nil; {
		v := e.Value.(*TimeTask)
		if v.Id == id {
			tc.data.Remove(e)
			return nil
		}
		e = e.Next()
	}
	return errors.New("del task fail")
}

//删除任务
func delTc(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	if len(req.Form["id"]) < 0 {
		fail(&w)
	}
	id, _ := strconv.Atoi(req.Form["id"][0])
	err := DelTc(id)
	if err != nil {
		fail(&w)
	}
	success(&w)
}

//获取状态
func status(w http.ResponseWriter, req *http.Request) {
	var ts = GetAllTasks()
	b, _ := json.Marshal(ts)
	w.Write(b)
}
