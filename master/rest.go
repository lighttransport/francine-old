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
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

/**
 * @apiDefinePermission LTE rights needed.
 * Optionally you can write here further Informations about the permission.
 *
 * An "apiDefinePermission"-block can have an "apiVersion", so you can attach the block to a specific version.
 *
 * @apiVersion v0
 */

/**
 * @api {post} /sessions Create new session
 * @apiVersion v0
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

	conn.Send("MULTI")
	conn.Send("SADD", "session", result.SessionId)
	conn.Send("SET", "session:"+result.SessionId+":modified", strconv.FormatInt(time.Now().Unix(), 10))
	conn.Send("SET", "session:"+result.SessionId+":input-json", requestJson.InputJson)
	if _, err := conn.Do("EXEC"); err != nil {
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

func doesSessionExist(session string, conn redis.Conn) (bool, error) {
	res, err := conn.Do("EXISTS", "session:"+session+":input-json")
	if err != nil {
		return false, err
	}

	if res.(int64) == 1 {
		return true, nil
	} else {
		return false, nil
	}
}

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

func deleteSession(session string, conn redis.Conn) error {
	members, err := conn.Do("SMEMBERS", "session:"+session+":resource")
	if err != nil {
		return err
	}
	_, err = conn.Do("DEL", "session:"+session+":input-json", "session:"+session+":resource",
		"session:"+session+":modified")
	if err != nil {
		return err
	}
	for _, memberBytes := range members.([]interface{}) {
		member := string(memberBytes.([]byte))

		conn.Send("MULTI")
		conn.Send("GET", "session:"+session+":resource:"+member)
		conn.Send("DEL", "session:"+session+":resource:"+member)
		conn.Send("SREM", "session", session)
		resp, err := conn.Do("EXEC")
		if err != nil {
			log.Printf("[MASTER] failed to delete %s\n", "session:"+session+":resource:"+member)
		}
		hash := string(resp.([]interface{})[0].([]byte))

		success := false
		for i := 0; i < 5; i++ {
			err = releaseResource(hash, conn)
			if err == nil {
				success = true
				break
			}
			log.Printf("[MASTER] retry deleting resource %s\n", hash)
			time.Sleep(200 * time.Microsecond)
		}
		if !success {
			log.Printf("[MASTER] failed to release resource %s\n", hash)
		}
	}
	return nil
}

/**
 * @api {delete} /sessions/:SessionId delete session
 * @apiVersion v0
 * @apiName NewSession
 * @apiGroup Render
 *
 * @apiSuccess {String} Status "OK" if success.
 *
 * @apiSuccessExample Success-Response:
 *     HTTP/1.1 200 OK
 *     {
 *       "Status": "Ok"
 *     }
 *
 */
func restDeleteSession(w http.ResponseWriter, r *http.Request, redisPool *redis.Pool, session string) {
	conn := redisPool.Get()
	defer conn.Close()
	var result struct {
		Status string
	}
	{
		e, err := doesSessionExist(session, conn)
		if err != nil {
			raiseHttpError(w, err)
			return
		}

		if e == false {
			result.Status = "SessionDoesNotExist"

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
	}

	if err := deleteSession(session, conn); err != nil {
		raiseHttpError(w, err)
		return
	}

	result.Status = "Ok"

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

/**
 * @api {put} /sessions/:sessionId/resources/:resourceName Add or update resource
 * @apiVersion v0
 * @apiName EditResource
 * @apiGroup Render
 *
 * @apiParam {binary} Input binary data. Saved as resourceName in the server.
 *
 * @apiSuccess {String} Status "OK" if success.
 * @apiSuccess {String} Name Filename of resource data.
 * @apiSuccess {String} Hash SHA256 hash value of resource data.
 * @apiSuccess {Number} Size of resource data in bytes.
 *
 * @apiSuccessExample Success-Response:
 *     HTTP/1.1 200 OK
 *     {
 *       "Status": "OK",
 *       "Name"  : "teapot.mesh",
 *       "Size"  : 1024,
 *       "Hash"  : "5968ad5c2a58c6ef057fb16387b4f02c0c297559043ed54e68432fffd01eb540",
 *     }
 *
 */
func restEditResource(w http.ResponseWriter, r *http.Request, redisPool *redis.Pool, session, resource string) {
	conn := redisPool.Get()
	defer conn.Close()

	var result struct {
		Status string
		Name   string
		Hash   string
		Size   int // Up to 2GB
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		raiseHttpError(w, err)
		return
	}

	{
		e, err := doesSessionExist(session, conn)
		if err != nil {
			raiseHttpError(w, err)
			return
		}

		if e == false {
			result.Status = "SessionDoesNotExist"

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
	}

	if verbose {
		log.Printf("[MASTER] putting resource %s (%d bytes)\n", resource, len(data))
	}

	hashBytes := sha256.Sum256(data)
	hash := hex.EncodeToString(hashBytes[:])

	prevHash, err := conn.Do("GET", "session:"+session+":resource:"+resource)
	if err != nil {
		raiseHttpError(w, err)
		return
	}

	if prevHash != nil {
		success := false
		for i := 0; i < 5; i++ {
			err = releaseResource(string(prevHash.([]byte)), conn)
			if err == nil {
				success = true
				break
			}
			time.Sleep(200 * time.Microsecond)
			if verbose {
				log.Printf("[MASTER] retry deleting resource %s\n", prevHash)
			}
		}
		if !success {
			if verbose {
				log.Printf("[MASTER] failed to release resource %s\n", prevHash)
			}
		}
	}

	conn.Send("MULTI")
	conn.Send("SET", "resource:"+hash, data)
	conn.Send("INCR", "resource:"+hash+":counter")
	conn.Send("SET", "session:"+session+":resource:"+resource, hash)
	conn.Send("SADD", "session:"+session+":resource", resource)
	conn.Send("SET", "session:"+session+":modified", strconv.FormatInt(time.Now().Unix(), 10))
	if _, err := conn.Do("EXEC"); err != nil {
		raiseHttpError(w, err)
		return
	}

	result.Status = "Ok"
	result.Name = resource
	result.Hash = hash
	result.Size = len(data)

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

/**
 * @api {patch} /sessions/:sessionId/resource Update resource
 * @apiVersion v0
 * @apiName UpdateResource
 * @apiGroup Render
 *
 * @apiParam {JSON} Input JSON patch.
 *
 * @apiSuccess {String} Status "OK" if success.
 *
 * @apiSuccessExample Success-Response:
 *     HTTP/1.1 200 OK
 *     {
 *       "Status": "OK",
 *     }
 *
 */
func restPatchResource(w http.ResponseWriter, r *http.Request, redisPool *redis.Pool, session string) {
	conn := redisPool.Get()
	defer conn.Close()

	var result struct {
		Status string
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		raiseHttpError(w, err)
		return
	}

	{
		e, err := doesSessionExist(session, conn)
		if err != nil {
			raiseHttpError(w, err)
			return
		}

		if e == false {
			result.Status = "SessionDoesNotExist"

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
	}

	if verbose {
		log.Printf("[MASTER] patching resource for session %s (%d bytes)\n", session, len(data))
	}

	// @todo {}
	//prevHash, err := conn.Do("GET", "session:"+session+":resource:"+resource)
	//if err != nil {
	//	raiseHttpError(w, err)
	//	return
	//}

	//if prevHash != nil {
	//	success := false
	//	for i := 0; i < 5; i++ {
	//		err = releaseResource(string(prevHash.([]byte)), conn)
	//		if err == nil {
	//			success = true
	//			break
	//		}
	//		time.Sleep(200 * time.Microsecond)
	//		if verbose {
	//			log.Printf("[MASTER] retry deleting resource %s\n", prevHash)
	//		}
	//	}
	//	if !success {
	//		if verbose {
	//			log.Printf("[MASTER] failed to release resource %s\n", prevHash)
	//		}
	//	}
	//}

	//conn.Send("MULTI")
	//conn.Send("SET", "resource:"+hash, data)
	//conn.Send("INCR", "resource:"+hash+":counter")
	//conn.Send("SET", "session:"+session+":resource:"+resource, hash)
	//conn.Send("SADD", "session:"+session+":resource", resource)
	//conn.Send("SET", "session:"+session+":modified", strconv.FormatInt(time.Now().Unix(), 10))
	//if _, err := conn.Do("EXEC"); err != nil {
	//	raiseHttpError(w, err)
	//	return
	//}

	result.Status = "Ok"

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
		*dst = image.NewRGBA64(src.Bounds())
		draw.Draw(*dst, (*dst).Bounds(), src, image.ZP, draw.Src)
	} else {
		m := &image.Uniform{&color.RGBA{0, 0, 0, 255 / uint8(ratio)}}
		draw.DrawMask(*dst, (*dst).Bounds(), src, src.Bounds().Min, m, m.Bounds().Min, draw.Over)
	}
	return
}

func clamp(f float32) uint16 {
	i := int32(f * 65535)
	if i < 0 {
		i = 0
	}

	return uint16(i)
}

func accumulateImage(dst *[]float32, src image.Image) {

	inBounds := src.Bounds()

	if *dst == nil {
		*dst = make([]float32, inBounds.Dx()*inBounds.Dy()*4)
	}

	var w = inBounds.Dx()

	for y := inBounds.Min.Y; y < inBounds.Max.Y; y++ {
		for x := inBounds.Min.X; x < inBounds.Max.X; x++ {
			r, g, b, a := src.At(x, y).RGBA()
			(*dst)[4*(y*w+x)+0] += float32(r) / float32(65535.0)
			(*dst)[4*(y*w+x)+1] += float32(g) / float32(65535.0)
			(*dst)[4*(y*w+x)+2] += float32(b) / float32(65535.0)
			(*dst)[4*(y*w+x)+3] += float32(a) / float32(65535.0)
		}
	}
}

func divImage(dst []float32, n float32) {

	invN := 1.0 / n

	for i := 0; i < len(dst); i++ {
		dst[i] *= invN
	}
}

/**
 * @api {post} /sessions/:sessionId/renders Run rendering
 * @apiVersion v0
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

// type Render struct {
// 	w              http.ResponseWriter
// 	r              *http.Request
// 	redisPool      *redis.Pool
// 	session        string
// 	renderTimes    int
// 	waitingDuration chan time.Duration
// }

func restNewRender(w http.ResponseWriter, r *http.Request, request chan RenderRequest, session string, renderTimes int) {
	// TODO: increment reference count of resources while renering is running

	res := make(chan Result, renderTimes)

	for i := 0; i < renderTimes; i++ {
		request <- RenderRequest{SessionId: session, ResultChan: res}
	}

	var accum []float32 = nil
	var bounds image.Rectangle

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

		// Assume all image have same extent
		bounds = curImg.Bounds()

		//composeImage(&composed, curImg, renderTimes)
		accumulateImage(&accum, curImg)
	}

	divImage(accum, float32(renderTimes))

	outimg := image.NewRGBA(bounds)

	var width = bounds.Dx()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r := accum[4*(y*width+x)+0]
			g := accum[4*(y*width+x)+1]
			b := accum[4*(y*width+x)+2]
			a := accum[4*(y*width+x)+3]
			rgba := color.RGBA64{clamp(r), clamp(g), clamp(b), clamp(a)}
			outimg.Set(x, y, rgba)
		}
	}

	var resBuf bytes.Buffer

	if err := jpeg.Encode(&resBuf, outimg, &jpeg.Options{Quality: 100}); err != nil {
		raiseHttpError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(resBuf.Bytes())

	return
}

func imax(x, y int) int {
	if x > y {
		return x
	} else {
		return y
	}
}
func imin(x, y int) int {
	if x > y {
		return y
	} else {
		return x
	}
}

func restHandler(path string, w http.ResponseWriter, r *http.Request, redisPool *redis.Pool, waitingDuration chan time.Duration, requestChan chan RenderRequest) {

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

	if matched := regexp.MustCompile("^/sessions/(.+)$").FindStringSubmatch(path); matched != nil {
		if r.Method == "DELETE" {
			if verbose {
				log.Println("[MASTER] request dispatched")
			}
			restDeleteSession(w, r, redisPool, matched[1])
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

	if matched := regexp.MustCompile("^/sessions/(.+)/resource$").FindStringSubmatch(path); matched != nil {
		if r.Method == "PATCH" {
			if verbose {
				log.Println("[MASTER] patch request dispatched")
			}
			restPatchResource(w, r, redisPool, matched[1])
			return
		}
	}

	if matched := regexp.MustCompile("^/sessions/(.+)/renders").FindStringSubmatch(path); matched != nil {
		if r.Method == "POST" {
			if verbose {
				log.Println("[MASTER] request dispatched")
			}
			m, _ := url.ParseQuery(r.URL.RawQuery)

			renderTimes := 1
			if m["parallel"] != nil {
				n, err := strconv.Atoi(m["parallel"][0])
				if err == nil {
					renderTimes = n
				}
			}

			renderTimes = imin(imax(renderTimes, 1), 256)

			if verbose {
				log.Printf("[MASTER] renderTimes = %d\n", renderTimes)
			}

			restNewRender(w, r, requestChan, matched[1], renderTimes)
			return
		}
	}

	log.Println("[MASTER] resource not found: " + path)
	http.Error(w, "resource not found", http.StatusNotFound)

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
	RenderId string
	Status   string
	Log      string
}

type Result struct {
	Err   error
	Ack   []byte
	Image []byte
}

type ResultReceiver struct {
	RenderId   string
	ResultChan chan Result
	BeginTime  time.Time
}

type RenderRequest struct {
	SessionId  string
	ResultChan chan Result
}

func readAllFromReceiver(receiver chan ResultReceiver, receivers *map[string]ResultReceiver) {
	for {
		select {
		case recvVal := <-receiver:
			(*receivers)[recvVal.RenderId] = recvVal
		default:
			return
		}
	}
}

func receiveRenderResult(receiver chan ResultReceiver, waitingDuration chan time.Duration, redisPool *redis.Pool) {
	conn := redisPool.Get()
	defer conn.Close()

	resultReceivers := make(map[string]ResultReceiver)

	for {
		if verbose {
			log.Println("[MASTER] waiting for lte-ack")
		}
		resp, err := conn.Do("BLPOP", "lte-ack", 0)
		if err != nil {
			log.Fatalln(err)
		}

		readAllFromReceiver(receiver, &resultReceivers)

		if resp != nil {
			lteAckBytes := resp.([]interface{})[1].([]byte)

			var lteAck LteAck
			err = json.Unmarshal(lteAckBytes, &lteAck)
			if err != nil {
				log.Println(err)
				continue
			}

			if verbose {
				log.Printf("[MASTER] lte-ack of type %s received\n", lteAck.Status)
			}

			receiver, ok := resultReceivers[lteAck.RenderId]
			if !ok {
				log.Println("[MASTER] couldn't match any result channels!")
				continue
			}

			delete(resultReceivers, lteAck.RenderId)

			switch lteAck.Status {
			case "Start":
				waitingDuration <- time.Now().Sub(receiver.BeginTime)

				resultReceivers[receiver.RenderId] = receiver

			case "Ok":
				imageDataResp, err := conn.Do("GET", "render_image:"+receiver.RenderId)
				if err != nil {
					receiver.ResultChan <- Result{Err: err}
					continue
				}

				_, err = conn.Do("DEL", "render_image:"+receiver.RenderId)
				if err != nil {
					receiver.ResultChan <- Result{Err: err}
					continue
				}

				var imageDataJson struct {
					JpegData string `json:"jpegdata"`
				}

				if err := json.Unmarshal(imageDataResp.([]byte), &imageDataJson); err != nil {
					receiver.ResultChan <- Result{Err: err}
					continue
				}

				imageData, err := base64.StdEncoding.DecodeString(imageDataJson.JpegData)
				if err != nil {
					receiver.ResultChan <- Result{Err: err}
					continue
				}

				receiver.ResultChan <- Result{Image: imageData}

			case "LinkError":
				receiver.ResultChan <- Result{Ack: lteAckBytes}

			default:
				receiver.ResultChan <- Result{Err: errors.New("unknown lte-ack status: " + lteAck.Status)}
			}

		}
	}
}
func dispatchRenderRequest(request *RenderRequest, conn redis.Conn) (string, error) {

	conn.Send("MULTI")
	conn.Send("GET", "session:"+request.SessionId+":input-json")
	conn.Send("SMEMBERS", "session:"+request.SessionId+":resource")
	conn.Send("SET", "session:"+request.SessionId+":modified", strconv.FormatInt(time.Now().Unix(), 10))
	redisResp, err := conn.Do("EXEC")
	if err != nil {
		return "", err
	}

	if redisResp == nil {
		return "", errors.New("result nil")
	}

	if redisResp.([]interface{})[0] == nil {
		return "", errors.New("input-json nil; might be deleted session")
	}

	message := Message{
		RenderId:  strconv.FormatInt(time.Now().UnixNano(), 10),
		SessionId: request.SessionId,
		InputJson: string(redisResp.([]interface{})[0].([]byte))}

	for _, resourceNameBytes := range redisResp.([]interface{})[1].([]interface{}) {
		resourceName := string(resourceNameBytes.([]byte))
		if resp, err := conn.Do("GET", "session:"+request.SessionId+":resource:"+resourceName); err != nil {
			return "", err
		} else {
			resourceHash := string(resp.([]byte))
			message.Resources = append(message.Resources, Resource{resourceName, resourceHash})
		}
	}

	conn.Send("MULTI")
	for _, resource := range message.Resources {
		conn.Send("INCR", "resource:"+resource.Hash+":counter")
	}
	if _, err := conn.Do("EXEC"); err != nil {
		return "", err
	}

	marshaled, err := json.Marshal(&message)
	if err != nil {
		return "", err
	}

	if _, err := conn.Do("RPUSH", "render-queue", marshaled); err != nil {
		return "", err
	}

	return message.RenderId, nil
}

func interactWithRedis(requestChan chan RenderRequest, waitingDuration chan time.Duration, redisPool *redis.Pool) {
	log.Println("[MASTER] init redis...")
	conn := redisPool.Get()
	defer conn.Close()
	if _, err := conn.Do("SETNX", "lte-counter", 0); err != nil {
		log.Println(err)
	}
	log.Println("[MASTER] initialized")

	receiver := make(chan ResultReceiver, 256)

	go receiveRenderResult(receiver, waitingDuration, redisPool)

	for {
		if verbose {
			log.Println("[MASTER] waiting for request")
		}

		request := <-requestChan

		if verbose {
			log.Println("[MASTER] request received!")
		}

		renderId, err := dispatchRenderRequest(&request, conn)
		if err != nil {
			request.ResultChan <- Result{Err: err}
			continue
		}

		if verbose {
			log.Println("[MASTER] dispatched and result receiver set")
		}

		receiver <- ResultReceiver{RenderId: renderId, ResultChan: request.ResultChan, BeginTime: time.Now()}
	}

}

func startRestServer(redisPool *redis.Pool, waitingDuration chan time.Duration) {
	requestChan := make(chan RenderRequest, 256)

	http.HandleFunc("/v0/", func(w http.ResponseWriter, r *http.Request) {
		restHandler(strings.TrimPrefix(r.URL.Path, "/v0"), w, r, redisPool, waitingDuration, requestChan)
	})

	go interactWithRedis(requestChan, waitingDuration, redisPool)

	http.ListenAndServe(":80", nil)
}
