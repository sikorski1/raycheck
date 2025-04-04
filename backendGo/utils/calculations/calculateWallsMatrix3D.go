package calculations

import (
	. "backendGo/types"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
)

// GeoJSON structures with flexible property types
type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}

type Feature struct {
	Type       string          `json:"type"`
	Properties map[string]any  `json:"properties"`
	Geometry   Geometry        `json:"geometry"`
	ID         string          `json:"id,omitempty"`
}

type Geometry struct {
	Type        string        `json:"type"`
	Coordinates [][][]float64 `json:"coordinates"`
}

func calculateWalls(folderPath string) string {
	rawPath := filepath.Join(folderPath, "rawBuildings.json")
	data, err := os.ReadFile(rawPath)
	if err != nil {
		log.Fatalf("Failed to read data file: %v", err)
	}
	var featureCollection FeatureCollection
	err = json.Unmarshal(data, &featureCollection)
	if err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}
	var buildings []Building
	for i, feature := range featureCollection.Features {
		buildingIndex := i + 1
		
		buildingName := fmt.Sprintf("Building %d", buildingIndex)
		if name, ok := feature.Properties["addr:housename"]; ok {
			buildingName = fmt.Sprintf("%v", name)
		}
		
		var heightInLevels float64 = 3
		if levels, ok := feature.Properties["building:levels"]; ok {
			switch v := levels.(type) {
			case float64:
				heightInLevels = v
			case int:
				heightInLevels = float64(v)
			case string:
				fmt.Sscanf(v, "%f", &heightInLevels)
			}
		}
		
		heightInMeters := heightInLevels * 3.0
		buildingOutput := Building{
			Name:   buildingName,
			Height: heightInMeters,
			Walls:  []Wall{},
		}
		if feature.Geometry.Type == "Polygon" {
			if len(feature.Geometry.Coordinates) > 0 {
				ring := feature.Geometry.Coordinates[0]
				for i := 0; i < len(ring); i++ {
				
					current := Point3D{
						X: ring[i][0],         
						Y: ring[i][1],          
						Z: heightInMeters,     
					}
					nextIdx := (i + 1) % len(ring)
					next := Point3D{
						X: ring[nextIdx][0],  
						Y: ring[nextIdx][1],   
						Z: heightInMeters,     
					}
					wall := Wall{Start: current, End: next}
					buildingOutput.Walls = append(buildingOutput.Walls, wall)
				}
			}
		}
		
		buildings = append(buildings, buildingOutput)
	}
	outputJSON, err := json.MarshalIndent(buildings, "", "  ")
	if err != nil {
		log.Fatalf("Error creating JSON: %v", err)
	}
	outputFilePath := filepath.Join(folderPath, "buildings.json")
	err = os.WriteFile(outputFilePath, outputJSON, 0644)
	if err != nil {
		log.Fatalf("Error writing file %s: %v", outputFilePath, err)
	}
	fmt.Printf("Saved all buildings to %s\n", outputFilePath)
	fmt.Println("Processing complete")
	return outputFilePath
}
func geoToMatrixIndex(lat, lon, latMin, latMax, lonMin, lonMax float64, size int) (int, int) {
	y := (lat - latMin) / (latMax - latMin) * float64(size-1)
	x := (lon - lonMin) / (lonMax - lonMin) * float64(size-1)
	i := int(math.Round(x))
	j := int(math.Round(y))
	return i, j
}

func drawLine(matrix [][][]float64, x1, y1, z1, x2, y2, z2, heightLevels, wallIndex, sizeX, sizeY int) {
	dx := x2 - x1
	dy := y2 - y1
	if z1 >= heightLevels {
		z1 = heightLevels - 1
	}
	if z2 >= heightLevels {
		z2 = heightLevels - 1
	}
	if x1 == x2 && x1 >= 0 && x1 < sizeX{
		if y1 > y2 {
			y1, y2 = y2, y1 
		}
		for y := y1; y <= y2; y++ {
			if y >= 0 && y < sizeY{
				for z := 0; z <= z1; z++ {
					if matrix[z][y][x1] >= 1000 && matrix[z][y][x1] != float64(1000 + wallIndex) {
						matrix[z][y][x1] = 10000 // Mark as corner
					} else {
						matrix[z][y][x1] = float64(1000 + wallIndex)
					}
				}
			}
		}
	} else if y1 == y2  && y1 >= 0 && y1 < sizeY {
		if x1 > x2 {
			x1, x2 = x2, x1
		}
		for x := x1; x <= x2; x++ {
			if x >= 0 && x < sizeX {
				for z := 0; z <= z1; z++ {
					if matrix[z][y1][x] >= 1000 && matrix[z][y1][x] != float64(1000 + wallIndex) {
						matrix[z][y1][x] = 10000 // Mark as corner
					} else {
						matrix[z][y1][x] = float64(1000 + wallIndex)
					}
				}
			}
		}
	} else {
		steps := int(math.Max(math.Abs(float64(dx)), math.Abs(float64(dy))))
		prevXIdx := x1
		prevYIdx := y1
		for j := 0; j <= steps; j++ {
			x := x1 + dx*j/steps
			y := y1 + dy*j/steps
			xIdx := x
			yIdx := y
			if prevXIdx < xIdx && prevYIdx < yIdx || prevXIdx < xIdx && prevYIdx > yIdx  {
				if yIdx >= 0 && yIdx < sizeY && prevXIdx >= 0 && prevXIdx < sizeX {
					for z := 0; z <= z1; z++ {
						if matrix[z][yIdx][prevXIdx] >= 1000 && matrix[z][yIdx][prevXIdx] != float64(1000 + wallIndex) {
							matrix[z][yIdx][prevXIdx] = 10000 // Mark as corner
						} else {
							matrix[z][yIdx][prevXIdx] = float64(1000 + wallIndex)
						}
					}
				}
			}
			if prevXIdx > xIdx && prevYIdx < yIdx  || prevXIdx > xIdx && prevYIdx > yIdx  {
				if xIdx >=0 && xIdx < sizeX && prevYIdx >= 0 && prevYIdx < sizeY {
					for z := 0; z <= z1; z++ {
						if matrix[z][prevYIdx][xIdx] >= 1000 && matrix[z][prevYIdx][xIdx] != float64(1000 + wallIndex) {
							matrix[z][prevYIdx][xIdx] = 10000 // Mark as corner
						} else {
							matrix[z][prevYIdx][xIdx] = float64(1000 + wallIndex)
						}
					}
				}
			} // walls continuity
			if xIdx >= 0 && xIdx < sizeX && yIdx >= 0 && yIdx < sizeY {
				for z := 0; z <= z1; z++ {
					if matrix[z][yIdx][xIdx] >= 1000 && matrix[z][yIdx][xIdx] != float64(1000 + wallIndex) {
						matrix[z][yIdx][xIdx] = 10000 // Mark as corner
					} else {
						matrix[z][yIdx][xIdx] = float64(1000 + wallIndex)
					}
				}
			}
			prevXIdx = xIdx
			prevYIdx = yIdx
		} 
	}
}

func calculateNormal3D( x1, y1, z1, x2, y2, z2 int) Normal3D {
	dx := x2 - x1
	dy := y2 - y1
	length := math.Hypot(float64(dx), float64(dy))
		if length == 0  {
			return Normal3D{Nx:0, Ny:0, Nz:0}
		}
		nx := -float64(dy)/length
		ny := float64(dx)/length
		return Normal3D{Nx:nx, Ny:ny, Nz:0}
}

func generateBuildingMatrix(buildings []Building, latMin, latMax, lonMin, lonMax float64, size, heightLevels int) ([][][]float64, []Normal3D){
	matrix := make([][][]float64, heightLevels)
	wallNormals := []Normal3D{}
	for z := range matrix {
		matrix[z] = make([][]float64, size)
		for y := range matrix[z] {
			matrix[z][y] = make([]float64, size)
			for x := range matrix[z][y] {
				matrix[z][y][x] = -150.0
			}
		}
	}
	wallsMapIndex := 0
	for _, building := range buildings {
		for _, wall := range building.Walls {
			i1, j1 := geoToMatrixIndex(wall.Start.Y, wall.Start.X, latMin, latMax, lonMin, lonMax, size)
			i2, j2 := geoToMatrixIndex(wall.End.Y, wall.End.X, latMin, latMax, lonMin, lonMax, size)			
			z1 := int(math.Round(wall.Start.Z))
			z2 := int(math.Round(wall.End.Z))
			normal := calculateNormal3D( i1, j1, z1, i2, j2, z2)
			if normal.Nx == 0 && normal.Ny == 0 {
				continue
			} 
			drawLine(matrix, i1, j1, z1, i2, j2, z2, heightLevels, wallsMapIndex, 250, 250)
			wallNormals = append(wallNormals, normal)
			wallsMapIndex++
		}
	}
	fmt.Printf("walls: %v \n", wallsMapIndex)
	fmt.Printf("normals: %v \n", len(wallNormals))
	return matrix, wallNormals
}


func saveBinary(data interface{}, folderPath, filename string) error {
	finalPath := filepath.Join(folderPath, filename)
	file, err := os.Create(finalPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := gob.NewEncoder(file)
	return encoder.Encode(data)
}

func CalculateWallsMatrix3D(folderPath string, mapConfig MapConfig) {
	buildingsFilePath := calculateWalls(folderPath)
	data, err := os.ReadFile(buildingsFilePath)
	if err != nil {
		log.Fatalf("Failed to read data file: %v", err)
	}
	var buildings []Building
	err = json.Unmarshal(data, &buildings)
	if err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}
	matrix, wallNormals := generateBuildingMatrix(buildings, mapConfig.LatMin, mapConfig.LatMax, mapConfig.LonMin, mapConfig.LonMax, mapConfig.Size, mapConfig.HeightMaxLevels)
	saveBinary(matrix, folderPath, "wallsMatrix3D.bin")
	saveBinary(wallNormals, folderPath, "wallNormals3D.bin")
}
func LoadMatrixBinary(path string, data interface{}) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := gob.NewDecoder(file)
	return decoder.Decode(data)
}