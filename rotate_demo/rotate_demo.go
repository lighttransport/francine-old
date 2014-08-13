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
	port            = "8080"
	parallel        = 1
	lowerBufferSize = 30
	upperBufferSize = 100
	maxQueue        = 100
)

var (
	lteHost  = ""
	modelDir = ""
	fps      = 10
)

var (
	replacedContent = ""
	replacedDst     = "scene/teapot_redis.json"
)

type Package struct {
	InputJson string
	Resources []Resource
}

type Resource struct {
	Src string
	Dst string
}

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

func registerSession(inputJson string) (string, error) {
	input := &NewSessionInput{InputJson: inputJson}
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

	return output.SessionId, nil
}

func initLTE(modelName string) (string, error) {
	modelPrefix := modelDir + "/" + modelName

	var model Package

	// TODO: check whether modelName included in models.json
	packageBytes, err := ioutil.ReadFile(modelPrefix + "/package.json")
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(packageBytes, &model)
	if err != nil {
		return "", err
	}

	sessionId, err := registerSession(model.InputJson)
	if err != nil {
		return "", err
	}

	for _, resource := range model.Resources {
		fileBytes, err := ioutil.ReadFile(modelPrefix + "/" + resource.Src)
		if err != nil {
			return "", err
		}

		err = putResource(sessionId, resource.Dst, fileBytes)
		if err != nil {
			return "", err
		}

		if resource.Dst == replacedDst {
			replacedContent = string(fileBytes)
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
	var omega float64 = 2.0 * math.Pi / (float64(fps) * 5.0)
	var theta float64 = 0.0

	queued := 0
	decrQueued := make(chan struct{}, 256)

	lastTime := time.Now()

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

		time.Sleep(time.Second/time.Duration(fps) - time.Now().Sub(lastTime))

		lastTime = time.Now()

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

	sessionId, err := initLTE("teapot")
	if err != nil {
		log.Println(err)
		return
	}

	stop := make(chan struct{})
	res := make(chan Result, 256)

	last := make(chan int)
	go issueRequests(sessionId, res, stop, last)

	buf := make(ResultSlice, 0)

	buffering := true

	beginTime := time.Now()
	passed := 0

	for {
		prev := len(buf)
		buf = obtainResultNonblocking(buf, res)
		cur := len(buf)
		passed += cur - prev

		if passed == 0 {
			time.Sleep(time.Second)
		} else {
			time.Sleep(time.Now().Sub(beginTime) / time.Duration(passed))
		}

		if len(buf) < lowerBufferSize {
			buffering = true
		}

		if len(buf) > upperBufferSize {
			buffering = false
		}

		if buffering {
			log.Printf("len of buf %d, buffering ...", len(buf))
			time.Sleep(time.Second * 1)
			continue
		}

		//websocket.Message.Receive(ws, nil)

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
	afterReplaced := []byte(strings.Replace(replacedContent,
		`    "eye" : [0.0, 20.0, 80.0],`,
		fmt.Sprintf(`    "eye": [%f, %f, %f],`, x, y, z), -1))

	err := putResource(sessionId, replacedDst, afterReplaced)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	modelDir = os.Getenv("MODEL_DIR")
	if modelDir == "" {
		log.Fatalln("please set MODEL_DIR")
	}

	lteHost = os.Getenv("LTE_HOST")
	if lteHost == "" {
		log.Fatalln("please set LTE_HOST")
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
