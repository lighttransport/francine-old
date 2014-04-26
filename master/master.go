package main

import (
	"github.com/garyburd/redigo/redis"
	"fmt"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"os"
	"log"
)

// instanceName = "lte-worker-" + time.Now().Format("20060102150405")

const (
	redisMaxIdle = 5
)

type config struct {
	RedisServer string
	ImageServer string
	TokenUrl    string
}

func getEtcdValue(etcdHost, key string) (string, error) {
	fmt.Println("[WORKER] etcd host: " + etcdHost)
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

	return parsed.Node.Value, nil
}

func getConfigFromEtcd(etcdHost string) (*config, error) {
	var err error
	c := new(config)
	if c.RedisServer, err = getEtcdValue(etcdHost, "redis-server"); err != nil {
		return nil, err
	}
	if c.TokenUrl, err = getEtcdValue(etcdHost, "token-url"); err != nil {
		return nil, err
	}
	return c, nil
}

func main() {
	etcdHost := os.Getenv("ETCD_HOST")

	if etcdHost == "" {
		fmt.Println("please set ETCD_HOST")
		os.Exit(1)
	}

	config, err := getConfigFromEtcd(etcdHost)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	http.Handle("/", http.FileServer(http.Dir("/tmp/lte_bin")))
	go func() {
		http.ListenAndServe(":8080", nil)
	}()

	redisPool := redis.NewPool(
		func() (redis.Conn, error) {
			return redis.Dial("tcp", config.RedisServer)
		}, redisMaxIdle)
	defer redisPool.Close()

	for {
		redisConn := redisPool.Get()
		resp, err := redisConn.Do("BLPOP", "worker-q", 10)
		if resp != nil {
		}

		if err != nil {
			redisConn.Close()
			log.Fatal(err)
		}

		redisConn.Close()
	}
}

