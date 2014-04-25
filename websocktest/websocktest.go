package main

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
	//"runtime/pprof"
)

const (
	sceneFrames = 100 // [1, 100)
	sceneFiles  = "/Users/peryaudo/Desktop/_scene/scene%05d.jpg"
	enableFps   = true
)

type Statistics struct {
	Prev       time.Time
	AvrNum     int
	AvrDen     int
	DataAvrNum int
	DataAvrDen int
}

func NewStatistics() *Statistics {
	s := new(Statistics)
	s.Prev = time.Now()
	return s
}

func (s *Statistics) Calc(contentLength int) {
	cur := time.Now()
	elp := cur.Sub(s.Prev)
	if elp == 0 {
		return
	}
	s.AvrNum = s.AvrNum + int(time.Second/elp)
	s.AvrDen++
	s.Prev = cur
	s.DataAvrNum = s.DataAvrNum + contentLength
	s.DataAvrDen = s.DataAvrDen + int(elp)
}

func (s *Statistics) GetFps() int {
	if s.AvrDen == 0 {
		return 0
	}
	return s.AvrNum / s.AvrDen
}

func (s *Statistics) GetKiloBytesPs() int {
	den := s.DataAvrDen / int(time.Second)
	if den == 0 {
		return 0
	}
	return (s.DataAvrNum / den) / 1024
}

func websockHandler(ws *websocket.Conn) {
	fmt.Fprintf(os.Stderr, "connection established\n")

	var scenes [][]byte
	for i := 0; i < sceneFrames; i++ {
		content, err := ioutil.ReadFile(fmt.Sprintf(sceneFiles, i+1))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			os.Exit(1)
		} else {
			scenes = append(scenes, content)
		}
	}

	stat := NewStatistics()

	for {
		for _, scene := range scenes {
			websocket.Message.Receive(ws, nil)
			if err := websocket.Message.Send(ws, scene); err != nil {
				fmt.Fprintf(os.Stderr, "connection dead\n")
				return
			}

			if enableFps {
				stat.Calc(len(scene))
				fmt.Fprintf(os.Stderr, "\r%d fps, %d KBytes/s.....", stat.GetFps(), stat.GetKiloBytesPs())
			}

		}
	}
}

func main() {
	//f, _ := os.Create("prof")
	//pprof.StartCPUProfile(f)

	http.Handle("/test", websocket.Handler(websockHandler))
	http.Handle("/", http.FileServer(http.Dir(".")))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
