package main

import (
	"bytes"
	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	ltePath         = "/bin/lte"
	redisMaxIdle    = 5
	lteAckTtl       = 3600 // one hour
	pingIntervalMin = 1    // minutes
)

func getEtcdValue(etcdHost, key string) (string, error) {
	resp, err := http.Get("http://" + etcdHost + "/v2/keys/" + key)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var parsed struct {
		Node struct {
			Value string `json:"value"`
		} `json:"node"`
	}

	json.Unmarshal(body, &parsed)

	log.Println("[WORKER] etcd host: " + etcdHost + ", value for " + key + " : " + parsed.Node.Value)

	return parsed.Node.Value, nil
}

type Resource struct {
	Name string
	Hash string
}

type Message struct {
	RenderId  string
	SessionId string
	InputJson string
	Resources []Resource
}

type LteAck struct {
	Status string
	Log    string
}

func kickRenderer(msgBytes []byte, redisPool *redis.Pool, redisHost string, redisPort string) {
	conn := redisPool.Get()
	defer conn.Close()

	var message Message

	json.Unmarshal(msgBytes, &message)

	resourceDir := "/tmp/renders/" + message.RenderId

	if err := os.MkdirAll(resourceDir, 0755); err != nil {
		log.Println(err)
		return
	}

	if err := os.Chdir(resourceDir); err != nil {
		log.Println(err)
		return
	}

	// write the resource files
	for _, resource := range message.Resources {
		data, err := conn.Do("GET", "resource:"+resource.Hash)
		if err != nil {
			log.Println(err)
			return
		}

		// TODO: it has obvious security problem! be aware!
		absResourcePath := resourceDir + "/" + resource.Name
		if err = os.MkdirAll(filepath.Dir(absResourcePath), 0755); err != nil {
			log.Println(err)
			return
		}

		file, err := os.Create(absResourcePath)
		if err != nil {
			log.Println(err)
			return
		}
		file.Write(data.([]byte))
		file.Close()
	}

	// do link check
	linkCheckCmd := exec.Command(ltePath, "--linkcheck", "--session="+message.RenderId,
		"--resource_basepath="+resourceDir,
		"--redis_host="+redisHost, "--redis_port="+redisPort,
		"-c", resourceDir+"/"+message.InputJson)
	log.Printf("[WORKER] exec: %+v\n", linkCheckCmd.Args)
	var linkCheckOutput bytes.Buffer
	linkCheckCmd.Stderr = &linkCheckOutput
	linkCheckCmd.Stdout = &linkCheckOutput

	if err := linkCheckCmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			sendLteAck(&LteAck{Status: "LinkError", Log: linkCheckOutput.String()}, message.RenderId, redisPool)
			return
		} else {
			log.Fatalln(err)
		}
	}

	rendererCmd := exec.Command(ltePath, "--session="+message.RenderId,
		"--resource_basepath="+resourceDir,
		"--redis_host="+redisHost, "--redis_port="+redisPort,
		resourceDir+"/"+message.InputJson)
	var rendererStderr bytes.Buffer
	var rendererStdout bytes.Buffer
	rendererCmd.Stdout = &rendererStdout
	rendererCmd.Stderr = &rendererStderr

	rendererErr := rendererCmd.Run()

	if rendererStdout.Len() > 0 {
		log.Println("lte:stdout: " + rendererStdout.String())
	}
	if rendererStderr.Len() > 0 {
		log.Println("lte:stderr: " + rendererStderr.String())
	}

	if rendererErr != nil {
		log.Println(rendererErr)
	}

	sendLteAck(&LteAck{Status: "Ok"}, message.RenderId, redisPool)
}

func sendLteAck(data *LteAck, renderId string, redisPool *redis.Pool) {
	conn := redisPool.Get()
	defer conn.Close()

	key := "lte-ack:" + renderId
	strData, _ := json.Marshal(data)

	conn.Send("MULTI")
	conn.Send("DEL", key)
	conn.Send("LPUSH", key, strData)
	conn.Send("EXPIRE", key, lteAckTtl)
	conn.Send("LTRIM", key, 0, 0)
	_, err := conn.Do("EXEC")

	if err != nil {
		log.Println(err)
	}

	log.Println("[WORKER] lte-ack end with " + data.Status)
}

func sendPings(workerName string, redisPool *redis.Pool) {
	conn := redisPool.Get()
	defer conn.Close()

	for {
		conn.Do("RPUSH", "cmd:lte-master", "ping:"+workerName)
		time.Sleep(pingIntervalMin * time.Minute)
	}
}

func main() {
	etcdHost := os.Getenv("ETCD_HOST")
	if etcdHost == "" {
		log.Fatalln("please set ETCD_HOST")
	}

	workerName := os.Getenv("WORKER_NAME")
	if workerName == "" {
		log.Fatalln("please set WORKER_NAME")
	}

	redisUrl, err := getEtcdValue(etcdHost, "redis-server")
	if err != nil {
		log.Fatalln(err.Error())
	}
	redisUrlSplit := strings.Split(redisUrl, ":")
	redisHost, redisPort := redisUrlSplit[0], redisUrlSplit[1]

	redisPool := redis.NewPool(
		func() (redis.Conn, error) {
			return redis.Dial("tcp", redisUrl)
		}, redisMaxIdle)
	defer redisPool.Close()

	go sendPings(workerName, redisPool)

	cmdQueueName := "cmd:" + workerName

	for {
		redisConn := redisPool.Get()

		resp, err := redisConn.Do("BLPOP", "render-queue", cmdQueueName, 1)

		if resp != nil {
			listName := string(resp.([]interface{})[0].([]byte))
			popped := resp.([]interface{})[1].([]byte)

			switch listName {
			case "render-queue":
				go kickRenderer(popped, redisPool, redisHost, redisPort)
			case cmdQueueName:
				switch string(popped) {
				case "stop":
					redisConn.Close()
					os.Exit(0)
				case "restart":
					redisConn.Close()
					os.Exit(1)
				}
			}
		}

		if err != nil {
			redisConn.Close()
			log.Fatalln(err)
		}

		redisConn.Close()
	}
}
