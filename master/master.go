package main

import (
	"bytes"
	"code.google.com/p/goauth2/oauth"
	"encoding/base64"
	"encoding/json"
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
	verbose      = true
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

	log.Println("[MASTER] etcd host: " + etcdHost + ", value for " + key + " : " + parsed.Node.Value)

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
	tokenUrl, err := getEtcdValue(etcdHost, "lte-worker-url")
	if err != nil {
		log.Fatal(err)
	}
	gceOAuthToken, err := getEtcdValue(etcdHost, "gce-oauth-token")
	if err != nil {
		log.Fatal(err)
	}
	redisServer, err := getEtcdValue(etcdHost, "redis-server")
	if err != nil {
		log.Fatal(err)
	}
	fluentdServer, err := getEtcdValue(etcdHost, "fluentd-server")
	if err != nil {
		log.Fatal(err)
	}
	instanceName := "lte-worker-" + time.Now().Format("20060102150405")
	zone := "us-central1-a"
	machineType := "n1-standard-1"

	var cloudConfig string
	if r, err := ioutil.ReadFile("/tmp/cloud-config-worker.yaml"); err != nil {
		log.Fatal(err)
	} else {
		cloudConfig = string(r)
	}

	cloudConfig = strings.Replace(cloudConfig, "<hostname>", instanceName, -1)
	cloudConfig = strings.Replace(cloudConfig, "<lte_worker_url>", tokenUrl, -1)
	cloudConfig = strings.Replace(cloudConfig, "<redis_server>", redisServer, -1)
	cloudConfig = strings.Replace(cloudConfig, "<fluentd_server>", fluentdServer, -1)

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
		log.Println(res)
	}

	log.Println("[WORKER] waiting 60s for disk preparing...")
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
		// NOTE: no accessConfigs[] = will have no external internet access
		//"accessConfigs": []interface{}{map[string]string{
		//	"name": "External NAT",
		//	"type": "ONE_TO_ONE_NAT"}}}},
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
		log.Println(res)
	}
}

func main() {
	workers := make(map[string]time.Time)

	etcdHost := os.Getenv("ETCD_HOST")

	redisUrl := os.Getenv("REDIS_HOST")
	if redisUrl == "" {

		if etcdHost == "" {
			log.Println("please set ETCD_HOST")
			os.Exit(1)
		}

		var err error
		redisUrl, err = getEtcdValue(etcdHost, "redis-server")
		if err != nil {
			log.Fatal(err)
		}
	}

	redisPool := redis.NewPool(
		func() (redis.Conn, error) {
			return redis.Dial("tcp", redisUrl)
		}, redisMaxIdle)
	defer redisPool.Close()

	go startRestServer(redisPool)

	for {
		redisConn := redisPool.Get()
		resp, err := redisConn.Do("BLPOP", "cmd:lte-master", 0)
		if resp != nil {
			popped := string(resp.([]interface{})[1].([]byte))
			split := strings.Split(popped, ":")
			switch split[0] {
			case "create":
				if etcdHost == "" {
					log.Println("please set ETCD_HOST")
					os.Exit(1)
				}
				go createWorkerInstance(etcdHost)
			case "ping":
				workers[split[1]] = time.Now()
				log.Printf("[MASTER] ping from %s\n", split[1])
			case "restart_workers":
				for worker, _ := range workers {
					redisConn.Do("RPUSH", "cmd:"+worker, "restart")
				}
			}
		}

		if err != nil {
			redisConn.Close()
			log.Fatalf("%s; failed; exit\n", err.Error())
		}

		redisConn.Close()
	}
}
