package main

//这个seaver不应该在这里写、因为task的func 无法根据json解析获取、建议放到程序目录下、此处当成一个例子、需要自己根据更改
import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	task "github.com/congci/task-go"
)

type Task = task.Task
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

var mode task.TT
var port = flag.String("port", "0.0.0.0:8001", "port set")

func runDo1() {
	fmt.Print("hahah")
}

func main() {
	flag.Parse()

	//初始化
	mode = task.NewTask(task.TIMEWHEEL)

	//如果要数据初始化则可以 task.AddOnlyTask()

	var test = &Task{
		Cycle:       1000,
		Duration:    1,
		Create_time: time.Now().Unix(),
		Func:        runDo1,
	}

	var test2 = &Task{
		Cycle:       10,
		Duration:    1,
		Create_time: time.Now().Unix(),
		Func: func() {
			fmt.Print("2 run")
		},
		EndFunc: func(task.Chanl) {
			fmt.Print("2 end")
		},
	}

	var test3 = &Task{
		Cycle:       -1,
		Duration:    1,
		Create_time: time.Now().Unix(),
		Func: func() {
			fmt.Print("3 run")
		},
		EndFunc: func(task.Chanl) {
			fmt.Print("3 end")
		},
	}

	mode.AddOnlyTask(test)
	mode.AddOnlyTask(test2)
	mode.AddOnlyTask(test3)

	mode.Start()

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
	id, _ := strconv.Atoi(req.Form["id"][0])
	err := mode.DelTc(id)
	if err != nil {
		fail(&w)
	}
	success(&w)
}

//获取状态
func status(w http.ResponseWriter, req *http.Request) {
	var ts = mode.GetAllTasks()
	b, _ := json.Marshal(ts)
	w.Write(b)
}

//根据自己的状态设置对应的函数！！！！！！！！！！！！！！！、如果没有则不会执行任务
func setFunc(t *Task) {
	if t.TaskName == "" {
		t.Func = nil
		t.EndFunc = nil
	}
}
