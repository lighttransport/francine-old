package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/garyburd/redigo/redis"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

/**
 * @apiDefinePermission LTE rights needed.
 * Optionally you can write here further Informations about the permission.
 *
 * An "apiDefinePermission"-block can have an "apiVersion", so you can attach the block to a specific version.
 *
 * @apiVersion 0.9.0
 */

/**
 * @api {post} /sessions Create new session
 * @apiVersion 0.9.0
 * @apiName NewSession
 * @apiGroup Render
 *
 * @apiParam {InputJSON} Input JSON scene filename.
 *
 * @apiSuccess {String} SessionId Session ID.
 *
 * @apiSuccessExample Success-Response:
 *     HTTP/1.1 200 OK
 *     {
 *       "SessionId": "XXXXXXXX"
 *     }
 *
 */
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

/**
 * @api {put} /sessions/:sessionId/resources/:resourceName Add or update resource
 * @apiVersion 0.9.0
 * @apiName EditResource
 * @apiGroup Render
 *
 * @apiParam {binary} Input binary data. Saved as resourceName in the server.
 *
 * @apiSuccess {String} Status "OK" if success.
 * @apiSuccess {String} Name Filename of resource data.
 * @apiSuccess {String} Hash SHA256 hash value of resource data.
 *
 * @apiSuccessExample Success-Response:
 *     HTTP/1.1 200 OK
 *     {
 *       "Status": "OK",
 *       "Name"  : "teapot.mesh",
 *       "Hash"  : "5968ad5c2a58c6ef057fb16387b4f02c0c297559043ed54e68432fffd01eb540",
 *     }
 *
 */
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

type Result struct {
	Err   error
	Ack   []byte
	Image []byte
}

func generateRenderMessage(session string, redisPool *redis.Pool) (*Message, error) {
	conn := redisPool.Get()
	defer conn.Close()

	message := new(Message)

	conn.Send("MULTI")
	conn.Send("GET", "session:"+session+":input-json")
	conn.Send("SMEMBERS", "session:"+session+":resource")
	redisResp, err := conn.Do("EXEC")
	if err != nil {
		return nil, err
	}

	message.RenderId = strconv.FormatInt(time.Now().UnixNano(), 10)
	message.SessionId = session
	message.InputJson = string(redisResp.([]interface{})[0].([]byte))

	for _, resourceNameBytes := range redisResp.([]interface{})[1].([]interface{}) {
		resourceName := string(resourceNameBytes.([]byte))
		if resp, err := conn.Do("GET", "session:"+session+":resource:"+resourceName); err != nil {
			return nil, err
		} else {
			resourceHash := string(resp.([]byte))
			message.Resources = append(message.Resources, Resource{resourceName, resourceHash})
		}
	}

	return message, nil
}

func requestRender(message *Message, redisPool *redis.Pool, res chan Result) {
	conn := redisPool.Get()
	defer conn.Close()

	marshaled, err := json.Marshal(message)
	if err != nil {
		res <- Result{Err: err}
		return
	}

	if _, err := conn.Do("RPUSH", "render-queue", marshaled); err != nil {
		res <- Result{Err: err}
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
			res <- Result{Err: err}
			return
		}
	}

	var lteAck LteAck
	json.Unmarshal(lteAckBytes, &lteAck)

	switch lteAck.Status {
	case "Ok":
		imageDataResp, err := conn.Do("GET", "render_image:"+message.RenderId)
		if err != nil {
			res <- Result{Err: err}
			return
		}

		var imageDataJson struct {
			JpegData string `json:"jpegdata"`
		}

		if err := json.Unmarshal(imageDataResp.([]byte), &imageDataJson); err != nil {
			res <- Result{Err: err}
			return
		}

		imageData, err := base64.StdEncoding.DecodeString(imageDataJson.JpegData)
		if err != nil {
			res <- Result{Err: err}
			return
		}

		res <- Result{Image: imageData}
		return

	case "LinkError":
		res <- Result{Ack: lteAckBytes}
		return
	default:
		res <- Result{Err: errors.New("unknown lte-ack status: " + lteAck.Status)}
		return
	}
}

func composeImage(dst *draw.Image, src image.Image, ratio int) {
	/*
		out_A = src_A + dst_A(1 - src_A)
		out_r = src_r * src_A + dst_r * dst_A (1 - src_A) / out_A
		Thus,
		dst_A = 1
		out_A = 1
		out_r = src_r * src_A + dst_r * (1 - src_A)
	*/
	if *dst == nil {
		*dst = image.NewRGBA(src.Bounds())
		draw.Draw(*dst, (*dst).Bounds(), src, image.ZP, draw.Src)
	} else {
		m := &image.Uniform{&color.RGBA{0, 0, 0, 255 / uint8(ratio)}}
		draw.DrawMask(*dst, (*dst).Bounds(), src, src.Bounds().Min, m, m.Bounds().Min, draw.Over)
	}
	return
}

/**
 * @api {post} /sessions/:sessionId/renders Run rendering
 * @apiVersion 0.9.0
 * @apiName NewRender
 * @apiGroup Render
 *
 * @apiDescription Run rendering and wait until the rendering finishes. This API is blocking operation.
 *
 * @apiSuccess {Binary} JPEG file(binary stream).
 * @apiError {String} Status Currently always "LinkError".
 * @apiError {String} Log Detailed error log.
 *
 * @apiErrorExample Error-Response:
 *     HTTP/1.1 200 OK
 *     {
 *       "Status": "LinkError",
 *       "Log"   : "File not found: teapot.json"
 *     }
 *
 */

const renderTimes = 4

func restNewRender(w http.ResponseWriter, r *http.Request, redisPool *redis.Pool, session string) {
	// TODO: increment reference count of resources while renering is running

	res := make(chan Result, renderTimes)

	for i := 0; i < renderTimes; i++ {
		message, err := generateRenderMessage(session, redisPool)
		if err != nil {
			raiseHttpError(w, err)
		}
		go requestRender(message, redisPool, res)
	}

	var composed draw.Image = nil
	for i := 0; i < renderTimes; i++ {
		received := <-res

		if received.Err != nil {
			raiseHttpError(w, received.Err)
			return
		}

		if received.Ack != nil {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			w.Write(received.Ack)
			return
		}

		buf := bytes.NewBuffer(received.Image)
		curImg, err := jpeg.Decode(buf)
		if err != nil {
			raiseHttpError(w, err)
			return
		}

		composeImage(&composed, curImg, renderTimes)
	}

	var resBuf bytes.Buffer

	if err := jpeg.Encode(&resBuf, composed, &jpeg.Options{Quality: 100}); err != nil {
		raiseHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(resBuf.Bytes())

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
