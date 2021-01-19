package main

import (
  "log"
	"strconv"
	"net"
	"os"
	"time"
	"gocv.io/x/gocv"
)

func main() {
	// Set up video source
  videoPath := "vid.avi"
	video, err := gocv.VideoCaptureFile(videoPath)
	if err != nil {
		log.Printf("Error opening video capture file: %s\n", videoPath)
		return
	}
  defer video.Close()

	// Buffer for frame
	img := gocv.NewMat()
  defer img.Close()

	// Build connection to server
	connection, err := net.Dial("tcp", "localhost:27001")
	// connection, err := net.Dial("tcp", "3.84.55.178:27001")
	if err != nil {
		panic(err)
	}
	defer connection.Close()

	for {
		// Read one frame from video source
		if ok := video.Read(&img); !ok {
			log.Printf("Video closed: %v\n", videoPath)
			break
		}
		if img.Empty() {
			continue
		}

		// Get metadata for one frame (frame header)
		dims := img.Size()  // [240 320]
		data := img.ToBytes()
		frame_size := len(data)   // Frame total size is 230400 in this example
		mattype := int(img.Type())

		// Construct and send the frame header: width, height, mattype and data_size
		// TODO: do not need to parse to string
		// (1) width
		width := fillString(strconv.Itoa(dims[0]), 10)
		// (2) height
		height := fillString(strconv.Itoa(dims[1]), 10)
		// (3) mattype
		frame_size_str := fillString(strconv.Itoa(frame_size), 10)
		// (4) data_size
		mattype_str := fillString(strconv.Itoa(mattype), 10)

		t1 := time.Now()
		// Send frame header out
		connection.Write([]byte(width))
		connection.Write([]byte(height))
		connection.Write([]byte(frame_size_str))
		connection.Write([]byte(mattype_str))

		// Send data for one frame
		start := 0
		for {
			c, err := connection.Write(data[start:])
			if err != nil {
				log.Println(err)
				os.Exit(0)
      }
			start += c
			if start == frame_size {
        break  // all data in this frame has been sent out
      }
		}
		t2 := time.Now()

		// Read ACK from server: get ACK only after the processing has been finished
		resultBuffer := make([]byte, 2)
		connection.Read(resultBuffer)
		if string(resultBuffer) != "OK"{
			log.Println("Frame result wrong")
			break
		}
		t3 := time.Now()

		log.Println(t2.Sub(t1))
		log.Println(t3.Sub(t2))
		log.Println(t3.Sub(t1))
		log.Println("========")
	}
}

func fillString(retunString string, toLength int) string {
	for {
		lengtString := len(retunString)
		if lengtString < toLength {
			retunString = retunString + ":"
			continue
		}
		break
	}
	return retunString
}
