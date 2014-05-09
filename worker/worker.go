package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	ltePath      = "/bin/lte"
	redisMaxIdle = 5
	lteAckTtl    = 3600 // one hour
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

	fmt.Println("[WORKER] etcd host: " + etcdHost + ", value for " + key + " : " + parsed.Node.Value)

	return parsed.Node.Value, nil
}

func kickRenderer(msgBytes []byte, redisPool *redis.Pool, redisHost string, redisPort string) {
	var msg struct {
		SessionId string `json:"session_id"`
		ShaderId  int    `json:"shader_id"`
		Code      string `json:"code"`
	}

	json.Unmarshal(msgBytes, &msg)

	exeDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	writtenDir := exeDir + "/shaders/" + strconv.Itoa(msg.ShaderId)

	if err := os.MkdirAll(writtenDir, 0755); err != nil {
		fmt.Println(err.Error())
		return
	}

	// write shader file
	shaderFile, err := os.Create(writtenDir + "/shader.c")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	shaderFile.Write([]byte(msg.Code))
	shaderFile.Close()

	// do link check
	linkCheckCmd := exec.Command(ltePath, "--linkcheck", "--session="+msg.SessionId,
		"--resource_basepath="+writtenDir,
		"--resource_basepath=/home/default/scene",
		"--redis_host="+redisHost, "--redis_port="+redisPort,
		"-c", "scene/teapot_redis.json")
	fmt.Printf("[WORKER] exec: %+v\n", linkCheckCmd.Args)
	var linkCheckOutput bytes.Buffer
	linkCheckCmd.Stderr = &linkCheckOutput
	linkCheckCmd.Stdout = &linkCheckOutput

	if err := linkCheckCmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			sendLteAck(map[string]string{"code": "linkerr", "log": linkCheckOutput.String()},
				msg.SessionId, redisPool)
			return
		} else {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	}

	rendererCmd := exec.Command(ltePath, "--session="+msg.SessionId,
		"--resource_basepath="+writtenDir,
		"--resource_basepath=/home/default/scene",
		"--redis_host="+redisHost, "--redis_port="+redisPort,
		"scene/teapot_redis.json")
	var rendererStderr bytes.Buffer
	var rendererStdout bytes.Buffer
	rendererCmd.Stdout = &rendererStdout
	rendererCmd.Stderr = &rendererStderr

	rendererErr := rendererCmd.Run()

	if rendererStdout.Len() > 0 {
		fmt.Println("lte:stdout: " + rendererStdout.String())
	}
	if rendererStderr.Len() > 0 {
		fmt.Println("lte:stderr: " + rendererStderr.String())
	}

	if rendererErr != nil {
		fmt.Println(rendererErr.Error())
	}

	sendLteAck(map[string]string{"code": "ok"}, msg.SessionId, redisPool)
}

func sendLteAck(data map[string]string, sessionId string, redisPool *redis.Pool) {
	conn := redisPool.Get()
	defer conn.Close()

	key := "lte-ack:" + sessionId
	strData, _ := json.Marshal(data)

	conn.Send("MULTI")
	conn.Send("DEL", key)
	conn.Send("LPUSH", key, strData)
	conn.Send("EXPIRE", key, lteAckTtl)
	conn.Send("LTRIM", key, 0, 0)
	_, err := conn.Do("EXEC")

	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Printf("[WORKER]lte-ack end: session:%s data: %+v\n", sessionId, data)
}

func main() {
	etcdHost := os.Getenv("ETCD_HOST")

	if etcdHost == "" {
		fmt.Println("please set ETCD_HOST")
		os.Exit(1)
	}

	redisUrl, err := getEtcdValue(etcdHost, "redis-server")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	redisUrlSplit := strings.Split(redisUrl, ":")
	redisHost, redisPort := redisUrlSplit[0], redisUrlSplit[1]

	redisPool := redis.NewPool(
		func() (redis.Conn, error) {
			return redis.Dial("tcp", redisUrl)
		}, redisMaxIdle)
	defer redisPool.Close()

	for {
		redisConn := redisPool.Get()

		resp, err := redisConn.Do("BLPOP", "render-q", 1)

		if resp != nil {
			go kickRenderer(resp.([]interface{})[1].([]byte), redisPool, redisHost, redisPort)
		}

		if err != nil {
			fmt.Println(err.Error())
			redisConn.Close()
			os.Exit(1)
		}

		redisConn.Close()
	}
}
