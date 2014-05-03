package main

import (
	"bytes"
	"code.google.com/p/goauth2/oauth"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	redisMaxIdle = 5
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

func postRequest(url string, data interface{}, client *http.Client) (string, error) {
	marshaled, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	buf := bytes.NewBuffer(marshaled)
	resp, err := client.Post(url, "application/json", buf)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func createWorkerInstance(etcdHost string) {
	tokenUrl, err := getEtcdValue(etcdHost, "token-url")
	if err != nil {
		log.Fatal(err)
	}
	gceOAuthToken, err := getEtcdValue(etcdHost, "gce-oauth-token")
	if err != nil {
		log.Fatal(err)
	}
	instanceName := "lte-worker-" + time.Now().Format("20060102150405")
	zone := "us-central1-a"
	machineType := "n1-standard-1"

	var cloudConfig string
	if r, err := ioutil.ReadFile("/tmp/cloud-config.yaml"); err != nil {
		log.Fatal(err)
	} else {
		cloudConfig = string(r)
	}

	cloudConfig = strings.Replace(cloudConfig, "<hostname>", instanceName, -1)
	cloudConfig = strings.Replace(cloudConfig, "<token_url>", tokenUrl, -1)

	decoded, err := base64.StdEncoding.DecodeString(gceOAuthToken)
	if err != nil {
		log.Fatal(err)
	}

	var transport oauth.Transport
	json.Unmarshal(decoded, &transport)

	if res, err := postRequest(`https://www.googleapis.com/compute/v1/projects/gcp-samples/zones/`+zone+`/disks?sourceImage=https%3A%2F%2Fwww.googleapis.com%2Fcompute%2Fv1%2Fprojects%2Fgcp-samples%2Fglobal%2Fimages%2Fcoreos-v282-0-0`,
		map[string]interface{}{
			"zone":        "https://www.googleapis.com/compute/v1/projects/gcp-samples/zones/" + zone,
			"name":        instanceName,
			"description": ""},
		transport.Client()); err != nil {
		log.Fatal(err)
	} else {
		fmt.Println(res)
	}

	fmt.Println("[WORKER] waiting 60s for disk preparing...")
	time.Sleep(60 * time.Second)

	req := map[string]interface{}{
		"disks": []interface{}{map[string]interface{}{
			"type":       "PERSISTENT",
			"boot":       true,
			"autoDelete": true,
			"mode":       "READ_WRITE",
			"deviceName": instanceName,
			"zone":       "https://www.googleapis.com/compute/v1/projects/gcp-samples/zones/" + zone,
			"source":     "https://www.googleapis.com/compute/v1/projects/gcp-samples/zones/" + zone + "/disks/" + instanceName}},
		"networkInterfaces": []interface{}{map[string]interface{}{
			"network": "https://www.googleapis.com/compute/v1/projects/gcp-samples/global/networks/lte-cluster"}},
		"metadata": map[string]interface{}{
			"items": []interface{}{
				map[string]string{
					"key":   "user-data",
					"value": cloudConfig}}},
		"zone":         "https://www.googleapis.com/compute/v1/projects/gcp-samples/zones/" + zone,
		"canIpForward": "false",
		"scheduling": map[string]interface{}{
			"automaticRestart":  true,
			"onHostMaintenance": "MIGRATE"},
		"machineType": "https://www.googleapis.com/compute/v1/projects/gcp-samples/zones/" + zone + "/machineTypes/" + machineType,
		"name":        instanceName,
		"serviceAccounts": []interface{}{
			map[string]interface{}{
				"email": "default",
				"scopes": []string{
					"https://www.googleapis.com/auth/userinfo.email",
					"https://www.googleapis.com/auth/compute",
					"https://www.googleapis.com/auth/devstorage.full_control"}}}}

	if res, err := postRequest(`https://www.googleapis.com/compute/v1/projects/gcp-samples/zones/`+zone+`/instances`,
		req, transport.Client()); err != nil {
		log.Fatal(err)
	} else {
		fmt.Println(res)
	}
}

func main() {
	etcdHost := os.Getenv("ETCD_HOST")

	if etcdHost == "" {
		fmt.Println("please set ETCD_HOST")
		os.Exit(1)
	}

	if len(os.Args) > 1 && os.Args[1] == "create" {
		createWorkerInstance(etcdHost)
		os.Exit(0)
	}

	/*redisPool := redis.NewPool(
		func() (redis.Conn, error) {
			redisServer, err := getEtcdValue(etcdHost, "redis-server")
			if err != nil {
				log.Fatal(err)
			}

			return redis.Dial("tcp", redisServer)
		}, redisMaxIdle)
	defer redisPool.Close()

	for {
		redisConn := redisPool.Get()
		resp, err := redisConn.Do("BLPOP", "worker-q", 1)
		if resp != nil {
			switch string(resp.([]interface{})[1].([]byte)) {
			case "create":
				go createWorkerInstance(etcdHost)
			}
		}

		if err != nil {
			redisConn.Close()
			fmt.Printf("%s; failed, wait 30 seconds and retry...\n", err.Error())
			time.Sleep(30 * time.Second)
		}

		redisConn.Close()
	}*/

	redisServer, err := getEtcdValue(etcdHost, "redis-server")
	if err != nil {
		log.Fatal(err)
	}

	for {
		redisConn, err := redis.Dial("tcp", redisServer)
		if err != nil {
			fmt.Printf("redis connection failed(%s), wait 30 seconds and retry...\n", err.Error())
			time.Sleep(30 * time.Second)
			continue
		}
		defer redisConn.Close()

		resp, err := redisConn.Do("BLPOP", "worker-q", 1)
		if resp != nil {
			switch string(resp.([]interface{})[1].([]byte)) {
			case "create":
				go createWorkerInstance(etcdHost)
			}
		}

		if err != nil {
			log.Fatal(err)
		}
		break
	}
}
