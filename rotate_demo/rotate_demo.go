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
	"strconv"
	"strings"
	"time"
)

const (
	port         = "8080"
	teapotPrefix = "../demo/scene"
	parallel     = 1
	fps          = 10
)

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

func websockHandler(ws *websocket.Conn) {
	log.Println("connection established")

	sessionId, err := initLTE()
	if err != nil {
		log.Println(err)
		return
	}

	stop := make(chan struct{})
	resultChan := make(chan []byte)

	go func() {
		var x, y, z float64 = 0.0, 20.0, 80.0
		var omega float64 = 2.0 * math.Pi / (float64(fps) * 10.0)
		var theta float64 = 0.0

		for {
			select {
			case <-stop:
				close(resultChan)
				break
			default:
				z = 80.0 * math.Cos(theta)
				x = 80.0 * math.Sin(theta)
				theta = math.Mod(theta+omega, 2.0*math.Pi)

				err = putRotate(sessionId, x, y, z)
				if err != nil {
					log.Fatalln(err)
				}

				time.Sleep(time.Second / fps)

				go func() {
					result, err := requestRender(sessionId)
					if err != nil {
						log.Fatalln(err)
					}
					resultChan <- result
				}()
			}
		}
	}()

	for result := range resultChan {
		if err != nil {
			continue
		}

		websocket.Message.Receive(ws, nil)

		err = websocket.Message.Send(ws, result)
		if err != nil {
			stop <- struct{}{}
		}

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

	log.Printf("lte host: %s\n", lteHost)
	http.Handle("/websock", websocket.Handler(websockHandler))
	http.Handle("/", http.FileServer(http.Dir(".")))
	log.Printf("running on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalln(err)
	}
}
