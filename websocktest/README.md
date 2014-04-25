# WebSocket benchmark

## Usage

    go run websocktest.go

## Config

Edit these lines in websocktest.go: 

    const (
    	sceneFrames = 100 // [1, 100)
    	sceneFiles  = "/Users/peryaudo/Desktop/_scene/scene%05d.jpg"
    	enableFps   = true
    )

the port is 8080

## How to convert a video into jpg or png

Do like this:

    ffmpeg -i video.mp4 -vcodec png "scene%05d.png"

