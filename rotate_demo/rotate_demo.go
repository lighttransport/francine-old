package main

import (
	"bytes"
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	port         = "8080"
	teapotPrefix = "../demo/scene"
	parallel     = 1
	bufferSize   = 30
	maxQueue     = 100
)

var fps = 10

var lteHost = ""

type NewSessionInput struct {
	InputJson string
}
type NewSessionOutput struct {
	SessionId string
}

func putResource(sessionId, name string, content []byte) error {
	req, err := http.NewRequest("PUT", "http://"+lteHost+"/sessions/"+sessionId+"/resources/"+name, bytes.NewBuffer(content))
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func initLTE() (string, error) {
	sessionId := ""
	{
		input := &NewSessionInput{InputJson: "scene/teapot_redis.json"}
		inputBytes, err := json.Marshal(input)
		if err != nil {
			return "", err
		}
		resp, err := http.Post("http://"+lteHost+"/sessions", "application/json", bytes.NewBuffer(inputBytes))
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		output := new(NewSessionOutput)
		err = json.Unmarshal(body, output)
		if err != nil {
			log.Printf("json: %s\n", string(body))
			return "", err
		}
		sessionId = output.SessionId
	}

	files := [][]string{
		[]string{"scene/teapot_redis.json", "teapot_redis.json"},
		[]string{"scene/teapot_scene.json", "teapot_scene.json"},
		[]string{"scene/teapot.material.json", "teapot.material.json"},
		[]string{"scene/shaders.json", "shaders.json"},
		[]string{"scene/teapot.mesh", "teapot.mesh"},
		[]string{"shader.c", "shader.c"},
		[]string{"shader.h", "shader.h"},
		[]string{"procedural-noise.c", "procedural-noise.c"},
		[]string{"light.h", "light.h"}}

	for _, file := range files {
		fileBytes, err := ioutil.ReadFile(teapotPrefix + "/" + file[1])
		if err != nil {
			return "", err
		}
		if err = putResource(sessionId, file[0], fileBytes); err != nil {
			return "", err
		}
	}

	log.Println("lte initialized")
	return sessionId, nil
}

func releaseLTE(sessionId string) error {
	req, err := http.NewRequest("DELETE", "http://"+lteHost+"/sessions/"+sessionId, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Println("lte released")
	return nil
}

func requestRender(sessionId string) ([]byte, error) {
	resp, err := http.Post("http://"+lteHost+"/sessions/"+sessionId+"/renders?parallel="+strconv.Itoa(parallel), "text/plain", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

type Result struct {
	Idx  int
	Data []byte
}

type ResultSlice []Result

func (rs ResultSlice) Len() int {
	return len(rs)
}

func (rs ResultSlice) Less(i, j int) bool {
	return rs[i].Idx < rs[j].Idx
}

func (rs ResultSlice) Swap(i, j int) {
	rs[i], rs[j] = rs[j], rs[i]
}

func issueRequests(sessionId string, res chan Result, stop chan struct{}, last chan int) {
	var x, y, z float64 = 0.0, 20.0, 80.0
	var omega float64 = 2.0 * math.Pi / (float64(fps) * 10.0)
	var theta float64 = 0.0

	queued := 0
	decrQueued := make(chan struct{}, 256)

	for i := 0; ; i++ {
		for {
			ok := false
			select {
			case <-stop:
				last <- (i - 1)
				return
			case <-decrQueued:
				queued--
			default:
				ok = true
			}
			if ok {
				break
			}
		}

		if queued >= maxQueue {
			log.Println("wait for relase of queue")
			time.Sleep(time.Second)
			continue
		}

		z = 80.0 * math.Cos(theta)
		x = 80.0 * math.Sin(theta)
		theta = math.Mod(theta+omega, 2.0*math.Pi)

		err := putRotate(sessionId, x, y, z)
		if err != nil {
			log.Fatalln(err)
		}

		time.Sleep(time.Second / time.Duration(fps))

		queued++

		go func(idx int) {
			data, err := requestRender(sessionId)
			if err != nil {
				log.Fatalln(err)
			}
			res <- Result{Idx: idx, Data: data}
			decrQueued <- struct{}{}
		}(i)
	}
}

func obtainResultNonblocking(buf ResultSlice, res chan Result) ResultSlice {
	changed := false
	for {
		select {
		case r := <-res:
			buf = append(buf, r)
			changed = true
		default:
			if changed {
				sort.Sort(buf)
			}
			return buf
		}
	}
}

func consumeRemaining(lastIdx int, res chan Result) {
	for r := range res {
		if r.Idx == lastIdx {
			return
		}
	}
}

func websockHandler(ws *websocket.Conn) {
	log.Println("connection established")

	sessionId, err := initLTE()
	if err != nil {
		log.Println(err)
		return
	}

	stop := make(chan struct{})
	res := make(chan Result, 256)

	last := make(chan int)
	go issueRequests(sessionId, res, stop, last)

	buf := make(ResultSlice, 0)

	for {
		buf = obtainResultNonblocking(buf, res)

		time.Sleep(time.Second * 4 / (time.Duration(fps) * 3))

		if len(buf) < bufferSize {
			log.Printf("len of buf %d, buffering ...", len(buf))
			time.Sleep(time.Second * 5)
			continue
		}

		websocket.Message.Receive(ws, nil)

		if err := websocket.Message.Send(ws, buf[0].Data); err != nil {
			log.Println(err)
			stop <- struct{}{}
			consumeRemaining(<-last, res)
			break
		}

		buf = buf[1:]
	}

	log.Println("disconnected; release LTE")
	err = releaseLTE(sessionId)
	if err != nil {
		log.Println(err)
	}

	return
}

func putRotate(sessionId string, x, y, z float64) error {
	mainJsonBytes, err := ioutil.ReadFile(teapotPrefix + "/teapot_redis.json")
	if err != nil {
		return err
	}
	mainJsonReplaced := []byte(strings.Replace(string(mainJsonBytes),
		`    "eye" : [0.0, 20.0, 80.0],`,
		fmt.Sprintf(`    "eye": [%f, %f, %f],`, x, y, z), -1))

	err = putResource(sessionId, "scene/teapot_redis.json", mainJsonReplaced)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("please pass lte host in the argument")
	}
	lteHost = os.Args[1]
	if len(os.Args) >= 3 {
		var err error
		fps, err = strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalln(err)
		}
	}
	log.Printf("fps set to be %d\n", fps)

	log.Printf("lte host: %s\n", lteHost)
	http.Handle("/websock", websocket.Handler(websockHandler))
	http.Handle("/", http.FileServer(http.Dir(".")))
	log.Printf("running on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalln(err)
	}
}
