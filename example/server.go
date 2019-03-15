package main

//这个seaver不应该在这里写、因为task的func 无法根据json解析获取、建议放到程序目录下、此处当成一个例子、需要自己根据更改
import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/rs/xid"

	task "github.com/congci/task-go"
)

type Task = task.Task
type Res struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

var failRes = Res{Code: 100, Msg: "fail"}
var successRes = Res{Code: 0, Msg: "success"}

//业务id关联、很多都是mysql的id、可能有的任务也关联了另一个任务、需要一起删除等
type Tidslink struct {
	rw   sync.RWMutex
	data map[string]string
}

var tidslink Tidslink

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

var mode task.TT
var port = flag.String("port", "0.0.0.0:8001", "port set")

func runDo1(*Task) {
	fmt.Print("hahah")
}

func main() {
	flag.Parse()

	//初始化
	mode = task.NewTask(task.TIMECHANNEL)

	//如果要数据初始化则可以 task.AddOnlyTask()

	var test = &Task{
		Tid:           "111",
		Cycle:         1000,
		Duration:      1,
		StartTaskTime: time.Now().Unix(),
		Func:          runDo1,
	}

	var test2 = &Task{
		Tid:           "2",
		Cycle:         10,
		Duration:      1,
		StartTaskTime: time.Now().Unix(),
		Func: func(*Task) {
			fmt.Print("2 run")
		},
		EndFunc: func(*Task, task.Chanl) {
			fmt.Print("2 end")
		},
	}

	var test3 = &Task{
		Tid:           "3",
		Cycle:         -1,
		Duration:      1,
		StartTaskTime: time.Now().Unix(),
		Func: func(*Task) {
			fmt.Print("3 run")
		},
		EndFunc: func(*Task, task.Chanl) {
			fmt.Print("3 end")
		},
	}

	mode.AddTc(test)
	mode.AddTc(test2)
	mode.AddTc(test3)

	go mode.Start()

	//接口都是业务型的
	http.HandleFunc("/pushsub", addTc)      //增加任务
	http.HandleFunc("/updatesub", updateTc) //更新任务
	http.HandleFunc("/status", status)      //获取状态
	http.HandleFunc("/delsub", delTc)       //删除任务
	err := http.ListenAndServe(*port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

//添加任务、添加任务和
func addTc(w http.ResponseWriter, req *http.Request) {
	b, _ := ioutil.ReadAll(req.Body)
	defer req.Body.Close()

	var task Task
	err := json.Unmarshal(b, &task)
	if err != nil {
		fail(&w)
		return
	}
	task.TaskStr = string(b)

	//业务id是0的话直接退出
	if task.Tid == "" {
		tid := xid.New().String()
		task.Tid = tid
		if task.Bid != "" {
			tidslink.rw.Lock()
			tidslink.data[task.Bid] = tid
			tidslink.rw.Unlock()
		}
	}
	//初始化task
	mode.AddTc(&task)
	success(&w)
}

//修改任务
func updateTc(w http.ResponseWriter, req *http.Request) {
	b, _ := ioutil.ReadAll(req.Body)
	defer req.Body.Close()

	var task Task
	err := json.Unmarshal(b, &task)
	if err != nil {
		fail(&w)
		return
	}
	task.TaskStr = string(b)

	if task.Bid != "" {
		tidslink.rw.Lock()
		tid, ok := tidslink.data[task.Bid]
		tidslink.rw.Unlock()
		if ok && task.Tid == "" {
			task.Tid = tid
		}
	}
	err = mode.UpdateTc(&task)
	if err != nil {
		fail(&w)
	}
	success(&w)
}

//删除任务
func delTc(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	if len(req.Form["id"]) < 0 {
		fail(&w)
	}
	id := req.Form["id"][0]

	//此处的id代表业务id、如果是任务id可以自己写
	tidslink.rw.Lock()
	tid, ok := tidslink.data[id]
	tidslink.rw.Unlock()
	if ok {
		err := mode.DelTc(tid)
		if err != nil {
			fail(&w)
			return
		}
	}

	success(&w)
}

//获取状态
func status(w http.ResponseWriter, req *http.Request) {
	var ts = mode.GetAllTasks()
	b, err := json.Marshal(ts)
	if err != nil {
		log.Print(err)
	}
	w.Write(b)
}
