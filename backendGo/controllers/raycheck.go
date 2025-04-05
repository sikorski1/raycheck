package controllers

import (
	"backendGo/utils/calculations"
	"backendGo/utils/raylaunching"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"github.com/gin-gonic/gin"
	"image/png"
	."backendGo/types"
	"time"
)

type MapConfiguration struct {
	Title string `json:"title"`
	Coordinates [][][]float64 `json:"coordinates"`
	Center [2]float64 `json:"center"`
	Bounds [2][2]float64 `json:"bounds"`
}

type Features struct {
	Type string `json:"type"`
	Properties interface{} `json:"properties"`
	Geometry interface{} `json:"geometry"`
	Id string `json:"id"`
}
type BuildingsConfiguration struct {
	Type string `json:"type"`
	Features []Features `json:"features"`
}

func GetMapConfiguration(context *gin.Context) {
	cwd, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	mapTitle := context.Param("mapTitle")
	fmt.Println(filepath.Join(cwd, "data", mapTitle,"mapData.json"))
	data, err := os.ReadFile(filepath.Join(cwd, "data", mapTitle,"mapData.json")) 
	if err != nil {
		log.Println("Failed to read data file")
		context.JSON(http.StatusInternalServerError, gin.H{"error":"Failed to read data file"})
		return
	}
	var mapData MapConfiguration
	err = json.Unmarshal(data, &mapData)
	if err != nil {
		log.Println("Failed to parse JSON")
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse JSON"})
		return
	}
	if mapData.Title == "" || len(mapData.Coordinates) == 0 {
		log.Println("Map configuration not found")
		context.JSON(http.StatusNotFound, gin.H{"error": "Map configuration not found"})
		return
	}

	context.JSON(http.StatusOK, mapData)
}

func GetBuildings(context *gin.Context) {
	cwd, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	mapTitle := context.Param("mapTitle")
	data, err := os.ReadFile(filepath.Join(cwd, "data", mapTitle,"rawBuildings.json"))
	if err != nil {
		log.Println("Failed to read data file")
		context.JSON(http.StatusInternalServerError, gin.H{"error":"Failed to read data file"})
	}
	var buildingData BuildingsConfiguration
	err = json.Unmarshal(data, &buildingData)

	if err != nil {
		log.Println("Failed to parse JSON")
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse JSON"})
		return
	}
	if len(buildingData.Features) == 0 {
		log.Println("Buildings configuration not found")
		context.JSON(http.StatusNotFound, gin.H{"error": "Buildings configuration not found"})
		return
	}
	context.JSON(http.StatusOK, buildingData)
}

func ComputeRays(context *gin.Context) {
	southwest :=  [2]float64{19.914029, 50.065311}
	southeast := [2]float64{19.917527, 50.065311}
	northeast := [2]float64{19.917527, 50.067556}
	numPoints := 500

	xStep := (southeast[0] - southwest[0]) / (float64(numPoints - 1));
	yStep := (northeast[1] - southeast[1]) / (float64(numPoints - 1));
	points := make([][2]float64, 0, numPoints*numPoints)
	for i := 0; i < numPoints; i++ {
		for j := 0; j < numPoints; j++ {
			x := southwest[0] + float64(i)*xStep
			y := southwest[1] + float64(j)*yStep
			points = append(points, [2]float64{x, y})
		}
	}
	context.JSON(http.StatusOK, points)

}

type RayLaunchRequest struct {
	StationHeight  float64 `json:"stationHeight"`
	Frequency      float64 `json:"freq"`
}

func Create3DRayLaunching(context *gin.Context) {
	mapTitle := context.Param("mapTitle")
	var request RayLaunchRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cwd, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	var matrix [][][]float64
	err = calculations.LoadMatrixBinary(filepath.Join(cwd, "data", mapTitle, "wallsMatrix3D.bin"), &matrix)
	if err != nil {
		log.Println("Failed to load matrix:", err)
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load matrix"})
		return
	}
	var wallNormals []Normal3D
	err = calculations.LoadMatrixBinary(filepath.Join(cwd, "data", mapTitle, "wallNormals3D.bin"), &wallNormals)
	if err != nil {
		log.Println("Failed to load matrix:", err)
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load matrix"})
		return
	}
	config := raylaunching.RayLaunching3DConfig{
		NumOfRaysAzim:        36,     
		NumOfRaysElev:        36,    
		NumOfInteractions:    4,     
		WallMapNumber:        1000,      
		CornerMapNumber:      10000,       
		SizeX:                250-1,    
		SizeY:                250-1,
		SizeZ:                30-1,     
		Step:                 1.0,    
		ReflFactor:           0.7,     
		TransmitterPower:     1.0,     
		MinimalRayPower:     -90.0,   
		TransmitterFreq:      2.4e9,   
		WaveLength:           0.125,  
	}
	start := time.Now()
	rayLaunching := raylaunching.NewRayLaunching3D(matrix, wallNormals,config)
	rayLaunching.CalculateRayLaunching3D()
	stop := time.Since(start)
	fmt.Printf("RayLaunching 3D calculation time: %v\n", stop)
	outputDir := filepath.Join("data", mapTitle, "imgs")
	err = os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		log.Fatalf("failed to create output directory: %v", err)
	}
	println(rayLaunching.PowerMap[0])
	for i := 0; i < int(config.SizeZ); i++ {
		heatmap := calculations.GenerateHeatmap(rayLaunching.PowerMap[i])
		filename := filepath.Join(outputDir, fmt.Sprintf("heatmap_%d.png", i))
		f, err := os.Create(filename)
		if err != nil {
			log.Printf("failed to create file %s: %v", filename, err)
			continue
		}
		err = png.Encode(f, heatmap)
		if err != nil {
			log.Printf("failed to encode image %s: %v", filename, err)
		}
		f.Close()
	}
	context.JSON(http.StatusOK, gin.H{
		"message":       "Request received successfully",
		"mapTitle":     mapTitle,
		"stationHeight": request.StationHeight,
		"frequency":     request.Frequency,
		"powerMap": rayLaunching.PowerMap,
	})
}