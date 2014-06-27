package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/garyburd/redigo/redis"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	ltePath         = "/bin/lte"
	redisMaxIdle    = 5
	lteAckTtl       = 3600 // one hour
	pingIntervalMin = 1    // minutes
	verbose         = false
	tmpPrefix       = "/tmp/lte"
	cleanupInterval = 10 // minutes
)

// TODO: DRY
func releaseResource(hash string, conn redis.Conn) error {
	if _, err := conn.Do("WATCH", "resource:"+hash, "resource:"+hash+":counter"); err != nil {
		return err
	}
	counterBytes, err := conn.Do("GET", "resource:"+hash+":counter")
	if err != nil {
		return err
	}

	counter, err := strconv.Atoi(string(counterBytes.([]byte)))
	if err != nil {
		return err
	}

	conn.Send("MULTI")
	if counter > 1 {
		conn.Send("SET", "resource:"+hash+":counter", counter-1)
	} else {
		conn.Send("DEL", "resource:"+hash, "resource:"+hash+":counter")
	}

	resp, err := conn.Do("EXEC")
	if err != nil {
		return err
	}
	if resp == nil {
		return errors.New("optimistic locking failed")
	}

	return nil
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
	RenderId string
	Status   string
	Log      string
}

func kickRenderer(msgBytes []byte, conn redis.Conn, redisHost string, redisPort string) {
	timeBeforeConn := time.Now()

	var message Message

	timeBeforeResource := time.Now()

	json.Unmarshal(msgBytes, &message)

	sendLteAck(&LteAck{RenderId: message.RenderId, Status: "Start"}, conn)

	resourceDir := tmpPrefix + "/renders/" + message.RenderId

	if err := os.MkdirAll(resourceDir, 0755); err != nil {
		log.Println(err)
		return
	}

	if err := os.MkdirAll(tmpPrefix+"/resources", 0755); err != nil {
		log.Println(err)
		return
	}

	if err := os.Chdir(resourceDir); err != nil {
		log.Println(err)
		return
	}

	// write the resource files
	for _, resource := range message.Resources {
		realPath := tmpPrefix + "/resources/" + resource.Hash
		if _, err := os.Stat(realPath); os.IsNotExist(err) {
			data, err := conn.Do("GET", "resource:"+resource.Hash)
			if err != nil {
				log.Println(err)
				return
			}

			file, err := os.Create(realPath)
			if err != nil {
				log.Println(err)
				return
			}

			if data == nil {
				log.Printf("[WORKER] cannot obtain resource %s\n", resource.Hash)
				return
			}

			file.Write(data.([]byte))
			file.Close()

			success := false
			for i := 0; i < 5; i++ {
				err = releaseResource(resource.Hash, conn)
				if err == nil {
					success = true
					break
				}
				log.Printf("[WORKER] retry deleting resource %s\n", resource.Hash)
				time.Sleep(200 * time.Microsecond)
			}
			if !success {
				log.Printf("[WORKER] failed to release resource %s\n", resource.Hash)
			}

		}

		// TODO: it has obvious security problem! be aware!
		symPath := resourceDir + "/" + resource.Name
		if err := os.MkdirAll(filepath.Dir(symPath), 0755); err != nil {
			log.Println(err)
			return
		}

		if err := os.Symlink(realPath, symPath); err != nil {
			log.Println(err)
			return
		}
	}

	timeBeforeRendering := time.Now()
	/*
		// do link check
		linkCheckCmd := exec.Command(ltePath, "--linkcheck", "--session="+message.RenderId,
			"--resource_basepath="+resourceDir,
			"--redis_host="+redisHost, "--redis_port="+redisPort,
			"-c", resourceDir+"/"+message.InputJson)
		if verbose {
			log.Printf("[WORKER] exec: %+v\n", linkCheckCmd.Args)
		}
		var linkCheckOutput bytes.Buffer
		linkCheckCmd.Stderr = &linkCheckOutput
		linkCheckCmd.Stdout = &linkCheckOutput

		if err := linkCheckCmd.Run(); err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				sendLteAck(&LteAck{RenderId: message.RenderId, Status: "LinkError", Log: linkCheckOutput.String()}, redisPool)
				return
			} else {
				log.Fatalln(err)
			}
		}
	*/
	parsed, _ := strconv.ParseInt(message.RenderId, 10, 64)
	seed := strconv.Itoa(int(parsed & (1<<30 - 1)))
	rendererCmd := exec.Command(ltePath, "--session="+message.RenderId,
		"--resource_basepath="+resourceDir,
		"--redis_host="+redisHost, "--redis_port="+redisPort,
		"--seed="+seed,
		resourceDir+"/"+message.InputJson)
	var rendererOutput bytes.Buffer
	rendererCmd.Stdout = &rendererOutput
	rendererCmd.Stderr = &rendererOutput

	rendererErr := rendererCmd.Run()

	if verbose {
		log.Println("[WORKER] lte: " + rendererOutput.String())
	}

	if rendererErr != nil {
		if _, ok := rendererErr.(*exec.ExitError); ok {
			sendLteAck(&LteAck{RenderId: message.RenderId, Status: "LinkError", Log: rendererOutput.String()}, conn)
		} else {
			log.Println(rendererErr)
		}
	}

	timeAfterEverything := time.Now()

	sendLteAck(&LteAck{RenderId: message.RenderId, Status: "Ok"}, conn)

	if err := os.RemoveAll(resourceDir); err != nil {
		log.Println(err)
		return
	}

	if verbose {
		log.Printf("[WORKER] conn: %d ms, pre: %d ms, render: %d ms",
			timeBeforeResource.Sub(timeBeforeConn).Nanoseconds()/1000/1000,
			timeBeforeRendering.Sub(timeBeforeResource).Nanoseconds()/1000/1000,
			timeAfterEverything.Sub(timeBeforeRendering).Nanoseconds()/1000/1000)
	}

	return
}

func sendLteAck(data *LteAck, conn redis.Conn) {
	strData, _ := json.Marshal(data)

	if _, err := conn.Do("RPUSH", "lte-ack", strData); err != nil {
		log.Println(err)
	}

	if verbose {
		log.Println("[WORKER] lte-ack end with " + data.Status)
	}
}

func sendPings(workerName string, redisPool *redis.Pool) {
	conn := redisPool.Get()
	defer conn.Close()

	for {
		conn.Do("RPUSH", "cmd:lte-master", "ping:"+workerName)
		time.Sleep(pingIntervalMin * time.Minute)
	}
}

func cleanResources(redisPool *redis.Pool) {
	conn := redisPool.Get()
	defer conn.Close()
	for {
		time.Sleep(cleanupInterval * time.Minute)
		log.Println("[WORKER] clean up unused resources ...")

		// list up files in resource directory
		files, err := ioutil.ReadDir(tmpPrefix + "/resources")
		if err != nil {
			log.Println(err)
			return
		}

		conn.Send("MULTI")
		for _, file := range files {
			conn.Send("EXISTS", "resource:"+file.Name())
		}

		exists, err := conn.Do("EXEC")
		if err != nil {
			log.Println(err)
			return
		}

		for i, exist := range exists.([]interface{}) {
			existBool := (exist.(int64) == 1)
			if existBool {
				continue
			}
			err = os.Remove(tmpPrefix + "/resources/" + files[i].Name())
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func main() {
	workerName := os.Getenv("WORKER_NAME")
	if workerName == "" {
		log.Fatalln("please set WORKER_NAME")
	}

	redisUrl := os.Getenv("REDIS_HOST")
	if redisUrl == "" {
		log.Fatalln("please set REDIS_HOST")
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

	go cleanResources(redisPool)

	for {
		redisConn := redisPool.Get()

		resp, err := redisConn.Do("BLPOP", "render-queue", cmdQueueName, 0)

		if resp != nil {
			listName := string(resp.([]interface{})[0].([]byte))
			popped := resp.([]interface{})[1].([]byte)

			switch listName {
			case "render-queue":
				kickRenderer(popped, redisConn, redisHost, redisPort)
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
