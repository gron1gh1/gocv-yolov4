package main

import (
	"fmt"
	"image"
	"image/color"
	"os"

	"gocv.io/x/gocv"
)

// getOutputsNames : YOLO Layer
func getOutputsNames(net *gocv.Net) []string {
	var outputLayers []string
	for _, i := range net.GetUnconnectedOutLayers() {
		layer := net.GetLayer(i)
		layerName := layer.GetName()
		if layerName != "_input" {
			outputLayers = append(outputLayers, layerName)
		}
	}
	return outputLayers
}

// PostProcess : All Detect Box
func PostProcess(frame gocv.Mat, outs *[]gocv.Mat) ([]image.Rectangle, []float32, []int) {
	var classIds []int
	var confidences []float32
	var boxes []image.Rectangle
	for _, out := range *outs {

		data, _ := out.DataPtrFloat32()
		for i := 0; i < out.Rows(); i, data = i+1, data[out.Cols():] {

			scoresCol := out.RowRange(i, i+1)

			scores := scoresCol.ColRange(5, out.Cols())
			_, confidence, _, classIDPoint := gocv.MinMaxLoc(scores)
			if confidence > 0.5 {

				centerX := int(data[0] * float32(frame.Cols()))
				centerY := int(data[1] * float32(frame.Rows()))
				width := int(data[2] * float32(frame.Cols()))
				height := int(data[3] * float32(frame.Rows()))

				left := centerX - width/2
				top := centerY - height/2
				classIds = append(classIds, classIDPoint.X)
				confidences = append(confidences, float32(confidence))
				boxes = append(boxes, image.Rect(left, top, width, height))
			}
		}
	}
	return boxes, confidences, classIds
}

// ReadCOCO : Read coco.names
func ReadCOCO() []string {
	var classes []string
	read, _ := os.Open("./assets/coco.names")
	defer read.Close()
	for {
		var t string
		_, err := fmt.Fscan(read, &t)
		if err != nil {
			break
		}
		classes = append(classes, t)
	}
	return classes
}

// drawRect : Detect Class to Draw Rect
func drawRect(img gocv.Mat, boxes []image.Rectangle, classes []string, classIds []int, indices []int) (gocv.Mat, []string) {
	var detectClass []string
	for _, idx := range indices {
		if idx == 0 {
			continue
		}
		gocv.Rectangle(&img, image.Rect(boxes[idx].Max.X, boxes[idx].Max.Y, boxes[idx].Max.X+boxes[idx].Min.X, boxes[idx].Max.Y+boxes[idx].Min.Y), color.RGBA{255, 0, 0, 0}, 2)
		gocv.PutText(&img, classes[classIds[idx]], image.Point{boxes[idx].Max.X, boxes[idx].Max.Y + 30}, gocv.FontHersheyPlain, 10, color.RGBA{0, 0, 255, 0}, 3)
		detectClass = append(detectClass, classes[classIds[idx]])
	}
	return img, detectClass
}

// Detect : Run YOLOv4 Process
func Detect(net *gocv.Net, src gocv.Mat, scoreThreshold float32, nmsThreshold float32, OutputNames []string, classes []string) (gocv.Mat, []string) {
	img := src.Clone()
	img.ConvertTo(&img, gocv.MatTypeCV32F)
	blob := gocv.BlobFromImage(img, 1/255.0, image.Pt(416, 416), gocv.NewScalar(0, 0, 0, 0), true, false)
	net.SetInput(blob, "")
	probs := net.ForwardLayers(OutputNames)
	boxes, confidences, classIds := PostProcess(img, &probs)

	indices := make([]int, 100)
	if len(boxes) == 0 { // No Classes
		return src, []string{}
	}
	gocv.NMSBoxes(boxes, confidences, scoreThreshold, nmsThreshold, indices)

	return drawRect(src, boxes, classes, classIds, indices)
}

// Process : Read Picture and Process
func Process(filename string) {
	// Init
	classes := ReadCOCO()

	net := gocv.ReadNet("./assets/yolov4.weights", "./assets/yolov4.cfg")
	defer net.Close()
	net.SetPreferableBackend(gocv.NetBackendType(gocv.NetBackendDefault))
	net.SetPreferableTarget(gocv.NetTargetType(gocv.NetTargetCPU))

	OutputNames := getOutputsNames(&net)

	window := gocv.NewWindow("yolo")
	img := gocv.IMRead(filename, gocv.IMReadColor)
	defer img.Close()

	detectImg, detectClass := Detect(&net, img.Clone(), 0.45, 0.5, OutputNames, classes)
	defer detectImg.Close()

	fmt.Printf("Dectect Class : %v\n", detectClass)
	window.IMShow(detectImg)
	gocv.IMWrite("result.jpg", detectImg)
	gocv.WaitKey(0)

}

func main() {
	Process(os.Args[1])
}
