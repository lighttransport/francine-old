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
	"strings"
)

const (
	etcdHost     = "http://127.17.42.1:4001"
	ltePath      = "/bin/lte"
	redisMaxIdle = 5
	lteAckTtl    = 3600 // one hour
)

type config struct {
	Node struct {
		Value string `json:"value"`
	} `json:"node"`
}

func getConfigFromEtcd() (*config, error) {
	fmt.Println("[WORKER] etcd host: " + etcdHost)
	resp, err := http.Get(etcdHost + "/v2/keys/redis-server")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var config config
	json.Unmarshal(body, &config)

	return &config, nil
}

func kickRenderer(msgBytes string, redisPool *redis.Pool, redisHost string, redisPort string) {
	var msg struct {
		SessionId string `json:"session_id"`
		ShaderId  string `json:"shader_id"`
		Code      string `json:"code"`
	}

	json.Unmarshal([]byte(msgBytes), &msg)

	exeDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	writtenDir := exeDir + "/shaders/" + msg.ShaderId

	if err := os.MkdirAll(writtenDir, 755); err != nil {
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
		"--resource_basepath="+writtenDir, "-c", "scene/teapot_redis.json")
	var linkCheckOutput bytes.Buffer
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
		"--redis_host="+redisHost, "--redis_port="+redisPort,
		"scene/teapot_redis.json")
	var rendererStderr bytes.Buffer
	rendererCmd.Stderr = &rendererStderr

	rendererErr := rendererCmd.Run()

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

	fmt.Println("lte-ack end")
}

func main() {
	config, err := getConfigFromEtcd()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	redisUrl := config.Node.Value
	redisUrlSplit := strings.Split(redisUrl, ":")
	redisHost, redisPort := redisUrlSplit[0], redisUrlSplit[1]

	redisPool := redis.NewPool(
		func() (redis.Conn, error) {
			return redis.Dial("tcp", redisUrl)
		}, redisMaxIdle)
	defer redisPool.Close()

	for {
		redisConn := redisPool.Get()

		if resp, err := redisConn.Do("BLPOP", "render-q", 1); err == nil {
			go kickRenderer(resp.(string), redisPool, redisHost, redisPort)
		}

		redisConn.Close()
	}
}
