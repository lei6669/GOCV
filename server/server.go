package main

import (
  "log"
	"image"
	"image/color"
	"strings"
	"net"
	"os"
	"io"
	"strconv"
	"gocv.io/x/gocv"
  "time"
)

func main() {
	server, err := net.Listen("tcp", "0.0.0.0:27001")
	if err != nil {
		log.Println("Error listetning: ", err)
		os.Exit(1)
	}
	defer server.Close()
	log.Println("Server started! Waiting for connections...")
	for {
		connection, err := server.Accept()
		if err != nil {
			log.Println("Error: ", err)
			os.Exit(1)
		}
		log.Println("Client connected")
		go objectDetectionHandler(connection)
	}
}

func objectDetectionHandler(connection net.Conn) {
	// Model setup
	model := "data/frozen_inference_graph.pb"
	config := "data/ssd_mobilenet_v1.pbtxt"
	backend := gocv.NetBackendDefault
	target := gocv.NetTargetCPU
	net := gocv.ReadNet(model, config)
	if net.Empty() {
		log.Fatalf("Error reading network model from : %v %v\n", model, config)
	}
	defer net.Close()
	net.SetPreferableBackend(gocv.NetBackendType(backend))
	net.SetPreferableTarget(gocv.NetTargetType(target))
	ratio := 1.0 / 127.5
	mean := gocv.NewScalar(127.5, 127.5, 127.5, 0)
	swapRGB := true

	// window := gocv.NewWindow("Object Detection")
	// defer window.Close()

	// buffer for frame
	img := gocv.NewMat()
	defer img.Close()
	// buffer for frame header
	width_str := make([]byte, 10)
	height_str := make([]byte, 10)
	frame_size_str := make([]byte, 10)
	mattype_str := make([]byte, 10)

	// Each iteration for one frame
	for {
		// Read the frame header
		// TODO: now only checked the error on frame data. Should check err and read bytes for all read and write
		_, err := connection.Read(width_str)
		if err == io.EOF {
			log.Println("EOF")
			break
		}
		t1 := time.Now()
		connection.Read(height_str)
		connection.Read(frame_size_str)
		connection.Read(mattype_str)
		width, _ := strconv.Atoi(strings.Trim(string(width_str), ":"))
		height, _ := strconv.Atoi(strings.Trim(string(height_str), ":"))
		frame_size, _ := strconv.ParseInt(strings.Trim(string(frame_size_str), ":"), 10, 64)  // int64: 10 is base-10, 64 is for int64
		mattype, _ := strconv.Atoi(strings.Trim(string(mattype_str), ":"))

		// Read data for one frame
		data := make([]byte, frame_size)
		start := 0
		for {
			c, err := connection.Read(data[start:])
			if err != nil {
				log.Println(err)
				os.Exit(0)
			}
			start += c
			if int64(start) == frame_size {  // all data in this frame has been read
        break
      }
		}
		t2 := time.Now()

		// Convert the frame from bytes to img mat
		img, err := gocv.NewMatFromBytes(width, height, gocv.MatType(mattype), data)
		if err != nil {
			log.Fatalf("Error converting bytes to matrix: %v", err)
		}
		// Process the frame
		blob := gocv.BlobFromImage(img, ratio, image.Pt(300, 300), mean, swapRGB, false)  // convert image Mat to 300x300 blob that the object detector can analyze
		net.SetInput(blob, "")  																													// feed the blob into the detector
		prob := net.Forward("")  																													// run a forward pass thru the network
		performDetection(&img, prob)
		prob.Close()
		blob.Close()
		t3 := time.Now()

		// show this frame
		// window.IMShow(img)
		// if window.WaitKey(1) >= 0 {
		// 	break
		// }

		log.Println(t2.Sub(t1))
		log.Println(t3.Sub(t2))
		log.Println(t3.Sub(t1))
		log.Println("========")

		// Send back ACK for this frame
		ACKinfo := "OK"
	  connection.Write([]byte(ACKinfo))
	}
	log.Println("Client done")
}

func performDetection(frame *gocv.Mat, results gocv.Mat) {
	for i := 0; i < results.Total(); i += 7 {
		confidence := results.GetFloatAt(0, i+2)
		if confidence > 0.5 {
			left := int(results.GetFloatAt(0, i+3) * float32(frame.Cols()))
			top := int(results.GetFloatAt(0, i+4) * float32(frame.Rows()))
			right := int(results.GetFloatAt(0, i+5) * float32(frame.Cols()))
			bottom := int(results.GetFloatAt(0, i+6) * float32(frame.Rows()))
			gocv.Rectangle(frame, image.Rect(left, top, right, bottom), color.RGBA{0, 255, 0, 0}, 2)
		}
	}
}
