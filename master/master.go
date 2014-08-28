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
	"strconv"
	"strings"
	"time"
)

const (
	//zone                  = "asia-east1-a"
	redisMaxIdle            = 5
	verbose                 = false
	zone                    = "us-central1-a"
	machineType             = "n1-highcpu-16"
	sessionTimeout          = 60 // minutes
	sessionCleanupIntereval = 10 // minutes
	instanceListInterval    = 2  // minutes
	instanceTimeout         = 3  // minutes
	instanceAdjustInterval  = 3  // minutes
	instanceAdjustNum       = 5  // instances
	instanceMax             = 10
	instanceMin             = 0
	instanceThresholdUpper  = 100 // ms
	instanceThresholdLower  = 20  // ms
	//sessionTimeout          = 1 // minutes
	//sessionCleanupIntereval = 2 // minutes
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

	if verbose {
		log.Println("[MASTER] etcd host: " + etcdHost + ", value for " + key + " : " + parsed.Node.Value)
	}

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

func getTransportFromToken(etcdHost string) (*oauth.Transport, error) {
	gceOAuthToken, err := getEtcdValue(etcdHost, "gce-oauth-token")
	if err != nil {
		return nil, err
	}

	decoded, err := base64.StdEncoding.DecodeString(gceOAuthToken)
	if err != nil {
		return nil, err
	}

	var transport oauth.Transport
	if err = json.Unmarshal(decoded, &transport); err != nil {
		return nil, err
	}

	return &transport, nil
}

func createWorkerInstances(etcdHost string, number int) {
	tokenUrl, err := getEtcdValue(etcdHost, "lte-worker-url")
	if err != nil {
		log.Fatal(err)
	}
	redisServer, err := getEtcdValue(etcdHost, "redis-server")
	if err != nil {
		log.Fatal(err)
	}
	logentriesToken, err := getEtcdValue(etcdHost, "logentries-token")
	if err != nil {
		log.Fatal(err)
	}

	transport, err := getTransportFromToken(etcdHost)
	if err != nil {
		log.Fatalln(err)
	}

	var cloudConfig string
	if r, err := ioutil.ReadFile("/tmp/cloud-config-worker.yaml"); err != nil {
		log.Fatal(err)
	} else {
		cloudConfig = string(r)
	}

	for i := 0; i < number; i++ {
		instanceName := "lte-worker-" + strings.Replace(time.Now().Format("20060102150405.000"), ".", "", -1)
		go createWorkerInstancesInternal(*transport, instanceName, tokenUrl, redisServer, logentriesToken, cloudConfig)
		time.Sleep(300 * time.Millisecond)
	}
}

const (
	DiskCreating = iota
	DiskFailed
	DiskReady
)

func getDiskState(transport oauth.Transport, diskName string) (int, error) {
	resp, err := transport.Client().Get(`https://www.googleapis.com/compute/v1/projects/gcp-samples/zones/` + zone + `/disks/` + diskName)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}

	var disk map[string]interface{}
	if err = json.Unmarshal(body, &disk); err != nil {
		return -1, err
	}

	switch disk["status"].(string) {
	case "RESTORING":
		fallthrough
	case "CREATING":
		return DiskCreating, nil
	case "FAILED":
		return DiskFailed, nil
	case "READY":
		return DiskReady, nil
	default:
		log.Fatalln("[MASTER] unknown disk state " + disk["status"].(string))
		return -1, nil
	}
}

func createWorkerInstancesInternal(transport oauth.Transport, instanceName, tokenUrl, redisServer, logentriesToken, cloudConfig string) {

	cloudConfig = strings.Replace(cloudConfig, "<hostname>", instanceName, -1)
	cloudConfig = strings.Replace(cloudConfig, "<lte_worker_url>", tokenUrl, -1)
	cloudConfig = strings.Replace(cloudConfig, "<redis_server>", redisServer, -1)
	cloudConfig = strings.Replace(cloudConfig, "<logentries_token>", logentriesToken, -1)

	if res, err := postRequest(`https://www.googleapis.com/compute/v1/projects/gcp-samples/zones/`+zone+`/disks?sourceImage=https%3A%2F%2Fwww.googleapis.com%2Fcompute%2Fv1%2Fprojects%2Fgcp-samples%2Fglobal%2Fimages%2Fcoreos-v282-0-0`,
		map[string]interface{}{
			"zone":        "https://www.googleapis.com/compute/v1/projects/gcp-samples/zones/" + zone,
			"name":        instanceName,
			"description": ""},
		transport.Client()); err != nil {
		log.Fatalln(err)
	} else {
		log.Println(res)
	}

	{
		i := 0
		for i = 0; i < 10; i++ {
			log.Println("[MASTER] waiting 30s for disk preparing...")
			time.Sleep(30 * time.Second)

			state, err := getDiskState(transport, instanceName)
			if err != nil {
				log.Fatalln(err)
			}

			if state == DiskFailed {
				log.Println("[MASTER] failed to create disk")
				return
			}
			if state == DiskReady {
				break
			}
		}
		if i >= 10 {
			log.Println("[MASTER] failed to create disk")
			return
		}
	}

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
			"network": "https://www.googleapis.com/compute/v1/projects/gcp-samples/global/networks/lte-cluster",
			// NOTE: no accessConfigs[] = will have no external internet access
			"accessConfigs": []interface{}{map[string]string{
				"name": "External NAT",
				"type": "ONE_TO_ONE_NAT"}}}},
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

func deleteWorkerInstance(etcdHost, instanceName string) error {
	transport, err := getTransportFromToken(etcdHost)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("DELETE", `https://www.googleapis.com/compute/v1/projects/gcp-samples/zones/`+zone+`/instances/`+instanceName, nil)
	if err != nil {
		return err
	}
	resp, err := transport.Client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)
	return nil
}

func stopWorker(workerName string, redisPool *redis.Pool) error {
	conn := redisPool.Get()
	defer conn.Close()

	_, err := conn.Do("RPUSH", "cmd:"+workerName, "stop")
	if err != nil {
		return err
	}

	return nil
}

func listWorkerInstances(etcdHost string) ([]string, error) {
	transport, err := getTransportFromToken(etcdHost)
	if err != nil {
		return nil, err
	}
	resp, err := transport.Client().Get(`https://www.googleapis.com/compute/v1/projects/gcp-samples/zones/` + zone + `/instances?filter=name%20eq%20%27.%2Alte-worker.%2A%27`)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var unmarshaled struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	}
	if err = json.Unmarshal(body, &unmarshaled); err != nil {
		return nil, err
	}

	res := make([]string, 0)

	for _, item := range unmarshaled.Items {
		res = append(res, item.Name)
	}

	return res, nil
}

type Worker struct {
	CreatedOn time.Time
	PingOn    time.Time
	Stopped   bool
}

func durMin(x, y time.Duration) time.Duration {
	if x > y {
		return y
	} else {
		return x
	}
}

func killZombies(etcdHost string, workers map[string]Worker) {
	now := time.Now()
	log.Println("[MASTER] start zombie hunting...")
	for name, info := range workers {
		createdDur := now.Sub(info.CreatedOn)
		pingDur := now.Sub(info.PingOn)
		log.Printf("[MASTER] %s created %d min before, ping %d min before\n", name, createdDur/time.Minute, pingDur/time.Minute)
		if durMin(createdDur, pingDur)/time.Minute > instanceTimeout {
			log.Printf("[MASTER] %s is zombie; going to delete ...\n", name)
			if err := deleteWorkerInstance(etcdHost, name); err != nil {
				log.Fatalln(err)
			}
		}
	}
	log.Println("[MASTER] finished zombie hunting.")
}

func manageWorkers(etcdHost string, redisPool *redis.Pool, workerPing chan string, waitingDuration chan time.Duration, reloadWorkers chan struct{}) {
	workers := make(map[string]Worker)

	workerListChan := make(chan []string, 8)

	reloadWorkerList := make(chan struct{}, 8)
	go func() {
		for {
			reloadWorkerList <- struct{}{}
			time.Sleep(instanceListInterval * time.Minute)
		}
	}()

	adjustInstance := make(chan struct{}, 8)
	go func() {
		for {
			time.Sleep(instanceAdjustInterval * time.Minute)
			adjustInstance <- struct{}{}
		}
	}()

	waitingDurNumer := 0
	waitingDurDenom := 0

	for {
		select {
		case workerName := <-workerPing:
			log.Printf("[MASTER] ping from %s\n", workerName)
			if worker, ok := workers[workerName]; ok {
				newWorker := worker
				newWorker.PingOn = time.Now()
				workers[workerName] = newWorker
			} else {
				log.Printf("[MASTER] unknown worker %s; ignore\n", workerName)
			}

		case waitingDurationVal := <-waitingDuration:
			waitingDurNumer += int(waitingDurationVal)
			waitingDurDenom += 1
			if verbose {
				log.Printf("[MASTER] waiting duration: %d ms\n", waitingDurationVal/time.Millisecond)
			}
		case <-reloadWorkers:
			redisConn := redisPool.Get()
			for workerName, _ := range workers {
				redisConn.Do("RPUSH", "cmd:"+workerName, "restart")
			}
			redisConn.Close()
		case <-reloadWorkerList:
			go func() {
				lst, err := listWorkerInstances(etcdHost)
				if err != nil {
					log.Fatalln(err)
				}
				workerListChan <- lst
			}()
		case workerList := <-workerListChan:
			newWorkers := make(map[string]Worker)
			for _, workerName := range workerList {
				prev, ok := workers[workerName]
				if ok {
					log.Printf("[MASTER] inherited previous worker info for %s\n", workerName)
					newWorkers[workerName] = prev
				} else {
					log.Printf("[MASTER] newly created worker %s detected\n", workerName)
					newWorkers[workerName] = Worker{CreatedOn: time.Now(), PingOn: time.Unix(0, 0)}
				}
			}
			log.Printf("[MASTER] %d workers found\n", len(workerList))
			workers = newWorkers
			if len(workers) == 0 {
				adjustInstance <- struct{}{}
			}
			go killZombies(etcdHost, workers)
		case <-adjustInstance:
			log.Printf("[MASTER] start automatic instance creation/deletion\n")
			log.Printf("[MASTER] available: %d workers\n", len(workers))
			newInstanceNum := len(workers)
			if waitingDurDenom == 0 {
				newInstanceNum -= instanceAdjustNum
				log.Println("[MASTER] no waiting duration log found")
			} else {
				waitingDurAvr := waitingDurNumer / waitingDurDenom
				log.Printf("[MASTER] average waiting duration: %d ms\n", waitingDurAvr/int(time.Millisecond))
				if waitingDurAvr > int(instanceThresholdUpper*time.Millisecond) {
					newInstanceNum += instanceAdjustNum
				} else if waitingDurAvr < int(instanceThresholdLower*time.Millisecond) {
					newInstanceNum -= instanceAdjustNum
				}
			}
			newInstanceNum = imax(instanceMin, imin(instanceMax, newInstanceNum))
			log.Printf("[MASTER] new instance number was decided to be %d\n", newInstanceNum)
			waitingDurDenom = 0
			waitingDurNumer = 0
			diff := newInstanceNum - len(workers)
			if diff > 0 {
				go createWorkerInstances(etcdHost, diff)
			} else {
				rem := -diff
				newWorkers := make(map[string]Worker)
				for name, info := range workers {
					newWorkers[name] = info
					if rem <= 0 {
						continue
					}
					if !info.Stopped {
						stopWorker(name, redisPool)
						info.Stopped = true
					}
					rem--
					newWorkers[name] = info
				}
				workers = newWorkers
			}
		}
	}
}

func cleanupSessions(redisPool *redis.Pool) {
	conn := redisPool.Get()
	defer conn.Close()

	for {
		time.Sleep(sessionCleanupIntereval * time.Minute)
		if verbose {
			log.Println("[MASTER] clean up unused sessions ...")
		}

		sessions, err := conn.Do("SMEMBERS", "session")
		if err != nil {
			log.Println(err)
			return
		}

		for _, session := range sessions.([]interface{}) {
			sessionString := string(session.([]byte))
			modified, err := conn.Do("GET", "session:"+sessionString+":modified")
			if err != nil {
				log.Println(err)
				return
			}
			if modified == nil {
				// FIXME: dirty fix
				continue
			}
			modifiedUnix, err := strconv.ParseInt(string(modified.([]byte)), 10, 64)
			if err != nil {
				log.Println(err)
				return
			}
			prev := time.Unix(modifiedUnix, 0)
			if time.Now().Sub(prev) > sessionTimeout*time.Minute {
				deleteSession(sessionString, conn)
			}
		}

	}
}

func main() {
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

	workerPing := make(chan string, 256)
	waitingDuration := make(chan time.Duration, 256)
	reloadWorkers := make(chan struct{}, 256)

	if etcdHost != "" {
		go manageWorkers(etcdHost, redisPool, workerPing, waitingDuration, reloadWorkers)
	}

	go startRestServer(redisPool, waitingDuration)

	go cleanupSessions(redisPool)

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
				number := 1
				if len(split) >= 2 {
					number, err = strconv.Atoi(split[1])
					if err != nil {
						log.Println("invalid number of created workers")
						continue
					}
				}
				go createWorkerInstances(etcdHost, number)
			case "ping":
				workerPing <- split[1]
			case "restart_workers":
				reloadWorkers <- struct{}{}
			}
		}

		if err != nil {
			redisConn.Close()
			log.Fatalf("%s; failed; exit\n", err.Error())
		}

		redisConn.Close()
	}
}
