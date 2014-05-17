package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/garyburd/redigo/redis"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

func restNewSession(w http.ResponseWriter, r *http.Request, redisPool *redis.Pool) {
	if verbose {
		log.Println("[MASTER] new session start")
	}
	conn := redisPool.Get()
	defer conn.Close()

	if verbose {
		log.Println("[MASTER] redis connected")
	}

	var requestJson struct {
		InputJson string
	}

	if reqBody, err := ioutil.ReadAll(r.Body); err != nil {
		raiseHttpError(w, err)
		return
	} else {
		if err := json.Unmarshal(reqBody, &requestJson); err != nil {
			raiseHttpError(w, err)
			return
		}
	}

	if verbose {
		log.Println("[MASTER] request read and parsed")
	}

	redisResp, err := conn.Do("INCR", "lte-counter")
	if err != nil {
		raiseHttpError(w, err)
		return
	}

	var result struct {
		SessionId string
	}

	if verbose {
		log.Println("[MASTER] incremented redis lte-counter")
	}

	result.SessionId = strconv.FormatInt(redisResp.(int64), 10)

	marshaled, err := json.Marshal(result)
	if err != nil {
		raiseHttpError(w, err)
		return
	}

	if verbose {
		log.Println("[MASTER] result marshaled")
	}

	if _, err := conn.Do("SET", "session:"+result.SessionId+":input-json", requestJson.InputJson); err != nil {
		raiseHttpError(w, err)
		return
	}

	if verbose {
		log.Println("[MASTER] wrote input-json to redis")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	w.Write(marshaled)

	if verbose {
		log.Println("[MASTER] sent all data, finished")
	}

	return
}

func restEditResource(w http.ResponseWriter, r *http.Request, redisPool *redis.Pool, session, resource string) {
	conn := redisPool.Get()
	defer conn.Close()

	// TODO: check whether the session exactly exists

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		raiseHttpError(w, err)
		return
	}

	log.Printf("[MASTER] putting resource %s (%d bytes)\n", resource, len(data))

	hashBytes := sha256.Sum256(data)
	hash := hex.EncodeToString(hashBytes[:])

	// TODO: use WATCH to safely and effectively perform these operations
	conn.Send("MULTI")
	conn.Send("SET", "resource:"+hash, data)
	conn.Send("INCR", "resource:"+hash+":counter")
	conn.Send("SET", "session:"+session+":resource:"+resource, hash)
	conn.Send("SADD", "session:"+session+":resource", resource)
	if _, err := conn.Do("EXEC"); err != nil {
		raiseHttpError(w, err)
		return
	}

	var result struct {
		Status string
		Name   string
		Hash   string
	}

	result.Status = "Ok"
	result.Name = resource
	result.Hash = hash

	marshaled, err := json.Marshal(result)
	if err != nil {
		raiseHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(marshaled)

	return
}

func raiseHttpError(w http.ResponseWriter, err error) {
	log.Println(err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return
}

// writing these definitions twice should be removed in the future

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

func restNewRender(w http.ResponseWriter, r *http.Request, redisPool *redis.Pool, session string) {
	conn := redisPool.Get()
	defer conn.Close()

	var message Message

	conn.Send("MULTI")
	conn.Send("GET", "session:"+session+":input-json")
	conn.Send("SMEMBERS", "session:"+session+":resource")
	redisResp, err := conn.Do("EXEC")
	if err != nil {
		raiseHttpError(w, err)
		return
	}

	message.RenderId = strconv.FormatInt(time.Now().UnixNano(), 10)
	message.SessionId = session
	//log.Printf("%+v\n", redisResp)
	message.InputJson = string(redisResp.([]interface{})[0].([]byte))

	for _, resourceNameBytes := range redisResp.([]interface{})[1].([]interface{}) {
		resourceName := string(resourceNameBytes.([]byte))
		if resp, err := conn.Do("GET", "session:"+session+":resource:"+resourceName); err != nil {
			raiseHttpError(w, err)
			return
		} else {
			resourceHash := string(resp.([]byte))
			message.Resources = append(message.Resources, Resource{resourceName, resourceHash})
		}
	}

	// TODO: increment reference count of resources while renering is running

	marshaled, err := json.Marshal(message)
	if err != nil {
		raiseHttpError(w, err)
		return
	}

	if _, err := conn.Do("RPUSH", "render-queue", marshaled); err != nil {
		raiseHttpError(w, err)
		return
	}

	var lteAckBytes []byte

	for {
		resp, err := conn.Do("BLPOP", "lte-ack:"+message.RenderId, 0)

		if resp != nil {
			lteAckBytes = resp.([]interface{})[1].([]byte)
			break
		}

		if err != nil {
			raiseHttpError(w, err)
			return
		}
	}

	var lteAck LteAck
	json.Unmarshal(lteAckBytes, &lteAck)

	switch lteAck.Status {
	case "Ok":
		imageDataResp, err := conn.Do("GET", "render_image:"+message.RenderId)
		if err != nil {
			raiseHttpError(w, err)
			return
		}

		var imageDataJson struct {
			JpegData string `json:"jpegdata"`
		}

		if err := json.Unmarshal(imageDataResp.([]byte), &imageDataJson); err != nil {
			raiseHttpError(w, err)
			return
		}

		imageData, err := base64.StdEncoding.DecodeString(imageDataJson.JpegData)
		if err != nil {
			raiseHttpError(w, err)
			return
		}

		if verbose {
			log.Printf("[MASTER] received image, send back to the client... (%d bytes base64 to %d bytes)\n", len(imageDataJson.JpegData), len(imageData))
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(imageData)
	case "LinkError":
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(lteAckBytes)
	default:
		raiseHttpError(w, errors.New("unknown lte-ack status: "+lteAck.Status))
	}

	return
}

func restHandler(w http.ResponseWriter, r *http.Request, redisPool *redis.Pool) {
	path := r.URL.Path

	if verbose {
		log.Println("[MASTER] rest request: " + path)
	}

	if regexp.MustCompile("^/sessions$").MatchString(path) {
		if r.Method == "POST" {
			if verbose {
				log.Println("[MASTER] request dispatched")
			}
			restNewSession(w, r, redisPool)
			return
		}
	}

	if matched := regexp.MustCompile("^/sessions/(.+)/resources/(.+)$").FindStringSubmatch(path); matched != nil {
		if r.Method == "PUT" {
			if verbose {
				log.Println("[MASTER] request dispatched")
			}
			restEditResource(w, r, redisPool, matched[1], matched[2])
			return
		}
	}

	if matched := regexp.MustCompile("^/sessions/(.+)/renders").FindStringSubmatch(path); matched != nil {
		if r.Method == "POST" {
			if verbose {
				log.Println("[MASTER] request dispatched")
			}
			restNewRender(w, r, redisPool, matched[1])
			return
		}
	}

	log.Println("[MASTER] resource not found: " + path)
	http.Error(w, "resource not found", http.StatusNotFound)

	return
}

func redisInit(redisPool *redis.Pool) {
	log.Println("[MASTER] init redis...")
	conn := redisPool.Get()
	defer conn.Close()
	if _, err := conn.Do("SETNX", "lte-counter", 0); err != nil {
		log.Println(err)
	}
	log.Println("[MASTER] initialized")
	return
}

func startRestServer(redisPool *redis.Pool) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		restHandler(w, r, redisPool)
	})

	go redisInit(redisPool)

	http.ListenAndServe(":80", nil)
}
