package main

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"
	//"runtime/pprof"
)

const (
	sceneFrameBegin = 5000 // inclusive
	sceneFrameEnd = 5100 // exclusive
	//sceneFiles  = "/Users/peryaudo/Desktop/_scene/scene%05d.jpg"
	sceneFiles  = "/Volumes/SSD/camera/2013_07_timelapse/IMG_%04d.JPG"
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
	for i := sceneFrameBegin; i < sceneFrameEnd; i++ {
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
			//websocket.Message.Receive(ws, nil)
			fmt.Printf("sending...\n")
			if err := websocket.Message.Send(ws, scene); err != nil {
				fmt.Fprintf(os.Stderr, "connection dead\n")
				return
			}
			fmt.Printf("sent\n")

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

	isServer := flag.Bool("server", false, "Server mode")

	flag.Parse()

	if (*isServer) {
		http.Handle("/test", websocket.Handler(websockHandler))
		http.Handle("/", http.FileServer(http.Dir(".")))
		fmt.Printf("running on 8080...")
		log.Fatal(http.ListenAndServe(":8080", nil))
	} else { // client
		config, err := websocket.NewConfig("ws://localhost:8080/test", "http://localhost:8080")
		if err != nil {
			panic(err)
		}

		sock, err := net.Dial("tcp", "localhost:8080")
		if err != nil {
			panic(err)
		}
		//switch e := err.(type) {
		//case *net.OpError:
		//	if e.Err == syscall.EADDRNOTAVAIL {
		//		continue
		//	}
		//}

		conn, err := websocket.NewClient(config, sock)
		if err != nil {
			panic(err)
		}

		for {
			var message string
			websocket.Message.Receive(conn, &message)
			//buf := make([]byte, 1024)
			//fmt.Printf("Conn!\n")

			//n, err := conn.Read(buf)
			//if err != nil {
			//	panic(err)
			//}
			fmt.Printf("len: %d\n", len(message))
		}
		conn.Close()
		
	}

}
