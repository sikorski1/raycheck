package raylaunching

import (
	. "backendGo/types"
	"fmt"
	"math"
	"math/cmplx"
)

type RayLaunching3DConfig struct {
	NumOfRaysAzim, NumOfRaysElev, NumOfInteractions, WallMapNumber, RoofMapNumber, CornerMapNumber int
	SizeX, SizeY, SizeZ, Step, ReflFactor, TransmitterPower, MinimalRayPower, TransmitterFreq, WaveLength float64
	TransmitterPos Point3D
	SingleRays []SingleRay
}

type RayPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
	Power    float64 `json:"power"`
}
type RayLaunching3D struct {
	PowerMap [][][]float64
	WallNormals []Normal3D
	Config RayLaunching3DConfig
	RayPaths [][]RayPoint  
}


func NewRayLaunching3D(matrix [][][]float64, wallNormals []Normal3D, config RayLaunching3DConfig) *RayLaunching3D {
	return &RayLaunching3D{
		PowerMap: matrix,
		WallNormals: wallNormals,
		Config: config,
		RayPaths: make([][]RayPoint, len(config.SingleRays)),
	}
}

func (rl *RayLaunching3D) CalculateRayLaunching3D() {
	for z := 0; z < int(rl.Config.TransmitterPos.Z); z++ {
		rl.PowerMap[z][int(rl.Config.TransmitterPos.Y)][int(rl.Config.TransmitterPos.X)] = 0
	}
	for i := 0; i < rl.Config.NumOfRaysAzim; i++ { // loop over horizontal dim
		theta := (2*math.Pi)/float64(rl.Config.NumOfRaysAzim)*float64(i) // from -π to π
		for j := 0; j < rl.Config.NumOfRaysElev; j++ { // loop over vertical dim
				
			var phi,dx,dy,dz float64

			// spherical coordinates
			if rl.Config.TransmitterPos.Z == 0 {
				// half sphere – from 0 to π/2
				phi = ((math.Pi/2) / float64(rl.Config.NumOfRaysElev)) *  float64(j) // from 0 to π/2
				dx = math.Cos(theta) * math.Cos(phi) * rl.Config.Step 
				dy = math.Sin(theta) * math.Cos(phi) * rl.Config.Step
				dz = math.Sin(phi) * rl.Config.Step
			} else {
				//full sphere – from 0 to π
				phi = math.Pi * float64(j) / float64(rl.Config.NumOfRaysElev) - math.Pi/2// from 0 to π
				dx = math.Cos(theta) * math.Cos(phi) * rl.Config.Step 
				dy = math.Sin(theta) * math.Cos(phi) * rl.Config.Step
				dz = math.Sin(phi) * rl.Config.Step
				
			}
			dx, dy, dz = math.Round(dx*1e15)/1e15, math.Round(dy*1e15)/1e15, math.Round(dz*1e15)/1e15

			/* getting past to next step,
			 omitting calculations for transmitter */

			x := rl.Config.TransmitterPos.X + dx
			y := rl.Config.TransmitterPos.Y + dy
			z := rl.Config.TransmitterPos.Z + dz

			targetRayIndex := rl.isTargetRay(i, j)

			// initial counters
			currInteractions := 0
			currPower := 0.0
			currWallIndex := 0
			currStartLengthPos := Point3D{X:rl.Config.TransmitterPos.X, Y:rl.Config.TransmitterPos.Y, Z:rl.Config.TransmitterPos.Z}
			currRayLength := 0.0
			currSumRayLength := 0.0
			currReflectionFactor := 1.0
			diffLossLdB:=0.0
			// main loop
			for (x >= 0 && x <= rl.Config.SizeX) && (y >= 0 && y < rl.Config.SizeY) && (z <= rl.Config.SizeZ) && currInteractions < rl.Config.NumOfInteractions && currPower >= rl.Config.MinimalRayPower {
				// reflection from the ground when z is below 0
				if (z < 0 && currWallIndex != rl.Config.RoofMapNumber) {
					dz = -dz
					currWallIndex = rl.Config.RoofMapNumber
					currInteractions++
					currSumRayLength += calculateDistance(currStartLengthPos, Point3D{X: x, Y: y, Z: z})
					currStartLengthPos = Point3D{X: x, Y: y, Z: z}
					nx, ny, nz := 0.0, 0.0, 1.0
					// calculate angle of incidence
					cosTheta := -(dx*nx + dy*ny + dz*nz)
					theta := math.Acos(cosTheta)
					currReflectionFactor *= calculateReflectionFactor(theta, "medium-dry-ground")
					z = 0
				}
				if (z < 0) {
					if (dz < 0) {
						dz = -dz	
					}
					z += dz
				}
				xIdx := int(math.Round(x/rl.Config.Step))
				yIdx := int(math.Round(y/rl.Config.Step))
				zIdx := int(math.Round(z/rl.Config.Step))
				index := int(rl.PowerMap[zIdx][yIdx][xIdx])
				// if (i==0 && j == 10) {
					// println("i:", i, "j:", j, "x:", xIdx, "y:", yIdx, "z:", zIdx, "index:", index,"currWallIndex:", currWallIndex, "dx:", dx, "dy:", dy, "dz:", dz)
				// }
				// reflection from the building roof
				if (index == rl.Config.RoofMapNumber) && currWallIndex != rl.Config.RoofMapNumber {
					dz = -dz
					currWallIndex = rl.Config.RoofMapNumber
					currInteractions++
					currSumRayLength += calculateDistance(currStartLengthPos, Point3D{X: x, Y: y, Z: z})
					currStartLengthPos = Point3D{X: x, Y: y, Z: z}
					nx, ny, nz := 0.0, 0.0, 1.0
					cosTheta := -(dx*nx + dy*ny + dz*nz)
					theta := math.Acos(cosTheta)
					currReflectionFactor *= calculateReflectionFactor(theta, "concrete")
					continue
				} 
				 if index == rl.Config.CornerMapNumber && currWallIndex != rl.Config.CornerMapNumber {
					currWallIndex = rl.Config.CornerMapNumber
					normals := getNeighborWallNormals(xIdx, yIdx, zIdx, rl)
					if len(normals) !=0 {
						maxTheta := -1.0
						bestNormal := normals[0]
						for k, n := range normals {
							//!!! MAP IS MIRRORED BY Y SO ALL Y NORMALS SHOULD BE MIRRORED !!!
								n.Ny = -n.Ny
								cosTheta := -(dx*n.Nx + dy*n.Ny + dz*n.Nz)
								theta := math.Acos(cosTheta)
								if (theta > maxTheta) {
									maxTheta = theta
									bestNormal = n 
									fmt.Printf("Promien %d, %d Normalna %d: Nx=%.3f, Ny=%.3f, Nz=%.3f Theta=%.3f\n", i, j, k, n.Nx, n.Ny, n.Nz,cosTheta)
								}
							} 
							fmt.Printf("Promien %d, %d, Max theta: %.3f,\n", i, j, maxTheta)
							if ( maxTheta >= math.Pi - 0.3) {
								break
							}
							thetaDeg := maxTheta * 180.0 / math.Pi
							println(thetaDeg)
							q90 := 0.3
							v := 3.5
							qj := math.Pow(thetaDeg/90.0*q90, v)

							// Illusory distance d1 = 1, d2 = 2 + qj (jak w dokumencie)
							
							d2 := 2 + qj

							// Oblicz stratę dyfrakcyjną
							diffLossLdB = 20 * math.Log10(4 * math.Pi * d2 / rl.Config.WaveLength)
							println("Strata dyf: ", diffLossLdB)
							// Oblicz współczynnik tłumienia w dziedzinie liniowej
							currInteractions++
							currSumRayLength += calculateDistance(currStartLengthPos, Point3D{X: x, Y: y, Z: z})
							dot := 2 * (dx*bestNormal.Nx  + dy*bestNormal.Ny + dz*bestNormal.Nz)
							dot = -dot
							// "Zginamy" promień w drugą stronę
							dx = dx - dot*bestNormal.Nx
							dy = dy - dot*bestNormal.Ny
							dz = dz - dot*bestNormal.Nz
							length := math.Sqrt(dx*dx + dy*dy + dz*dz)
							dx /= length
							dy /= length
							dz /= length
					}
					if len(normals) == 0 {
						break // brak ścian w pobliżu, nie da się dyfraktować
					}
					
				} 

				if index >= rl.Config.WallMapNumber && index < rl.Config.RoofMapNumber && index != currWallIndex + rl.Config.WallMapNumber{ 	// check if there is wall and if its diffrent from previous one
					currWallIndex = index - rl.Config.WallMapNumber

					//get wall normal
					nx, ny, nz := rl.WallNormals[currWallIndex].Nx, rl.WallNormals[currWallIndex].Ny, rl.WallNormals[currWallIndex].Nz

					//!!! MAP IS MIRRORED BY Y SO ALL Y NORMALS SHOULD BE MIRRORED !!!
					ny = -ny
					dot := 2 * (dx*nx + dy*ny + dz*nz)

					// calculate new direction
					dx = dx - dot*nx
					dy = dy - dot*ny
					dz = dz - dot*nz
					currInteractions++

					// sum distance and set new start position
					currSumRayLength += calculateDistance(currStartLengthPos, Point3D{X: x, Y: y, Z: z})
					currStartLengthPos = Point3D{X: x, Y: y, Z: z}
					cosTheta := -(dx*nx + dy*ny + dz*nz)
					if cosTheta > 1 {
						cosTheta = 1
					}
					if cosTheta < -1 {
						cosTheta = -1
					}
					theta := math.Acos(cosTheta)
					currReflectionFactor *= calculateReflectionFactor(theta, "concrete")
				} else {
					// calculate distance and transmittance
					currRayLength = calculateDistance(currStartLengthPos, Point3D{X: x, Y: y, Z: z}) + currSumRayLength

					H := calculateTransmittance(currRayLength, rl.Config.WaveLength, currReflectionFactor)
					currPower = 10*math.Log10(rl.Config.TransmitterPower) + 20*math.Log10(cmplx.Abs(H)) - diffLossLdB
					if (diffLossLdB > 0.0) {
						println("diffLoss: ",diffLossLdB)
						println("currPorwe: ",currPower)
					}
					// update power map if power is higher than previous one
					if rl.PowerMap[zIdx][yIdx][xIdx] == -150 || rl.PowerMap[zIdx][yIdx][xIdx] < currPower {
						rl.PowerMap[zIdx][yIdx][xIdx] = currPower
					} 
					if targetRayIndex >= 0 {
						rl.RayPaths[targetRayIndex] = append(rl.RayPaths[targetRayIndex], RayPoint{
							X: float64(math.Round(x/rl.Config.Step)), 
							Y: float64(math.Round(y/rl.Config.Step)), 
							Z: float64(math.Round(z/rl.Config.Step)), 
							Power: currPower,
						})
				}
				}
				// println("currReflectionFactor: ",currReflectionFactor)
				// update position
				x += dx
				y += dy
				z += dz
			}
		}
	}
}

func calculateDistance(p1, p2 Point3D) float64 {
	dist := math.Sqrt(math.Pow(p1.X-p2.X,2)+math.Pow(p1.Y-p2.Y,2)+math.Pow(p1.Z-p2.Z,2))
	return dist
}

func calculateTransmittance(r, waveLength, reflectionRef float64) complex128 {
	if r > 0 {
		H := complex(reflectionRef, 0) * complex(waveLength/(4*math.Pi*r), 0) *
			cmplx.Exp(complex(0,-2*math.Pi*r/waveLength)) 
		return H
	} else {
		return 0
	}
}

func (rl *RayLaunching3D) isTargetRay(i, j int) int {
	for idx, singleRay := range rl.Config.SingleRays {
		if i - singleRay.Azimuth == 0 && j - singleRay.Elevation == 0{
			return idx
		}
	}
	return -1 
}
func calculateReflectionFactor(angle float64, material string) float64 {
	if angle > math.Pi/2 {
		angle = math.Pi - angle
	}
	var eta float64;
	switch material {
		case "concrete":
			eta = 5.31 
		case "ceiling-board":
			eta = 1.50
		case "medium-dry-ground":
			eta = 15
		}
	sinTheta := math.Sin(angle)
	cosTheta := math.Cos(angle)
	if cosTheta > 1 {
    	cosTheta = 1
	}
	if cosTheta < -1 {
		cosTheta = -1
	}
	root := math.Sqrt(eta - sinTheta*sinTheta)
	R_TE := (cosTheta - root)/(cosTheta + root)
	R_TM := (eta*cosTheta - root)/(eta*cosTheta + root)
	reflectionFactor := (math.Pow(R_TE, 2) + math.Pow(R_TM, 2)) / 2
	// println("ANGLE: ", angle, "root: ", root,"R_TE: ", R_TE, " R_TM: ", R_TM, " reflectionFactor ", reflectionFactor)
	return reflectionFactor
}

func getNeighborWallNormals(x, y, z int, rl *RayLaunching3D) []Normal3D {
	neighborNormals := make(map[int]Normal3D)

	// Przeszukaj kostkę 3x3x3 wokół punktu (x,y,z)
	size :=3
	for dx := -size; dx <= size; dx++ {
		for dy := -size; dy <= size; dy++ {
			for dz := -size; dz <= size; dz++ {
				xprim := x + dx
				yprim := y + dy
				zprim := z + dz

				// sprawdź zakresy, by nie wyjść poza mapę
				if xprim < 0 || yprim < 0 || zprim < 0 || xprim >= int(rl.Config.SizeX )|| yprim >= int(rl.Config.SizeY) || zprim >= int(rl.Config.SizeZ) {
					continue
				}

				index := int(rl.PowerMap[zprim][yprim][xprim])
				if index >= rl.Config.WallMapNumber && index < rl.Config.RoofMapNumber {
					currWallIndex := index - rl.Config.WallMapNumber
					if _, exists := neighborNormals[currWallIndex]; !exists {
						neighborNormals[currWallIndex] = rl.WallNormals[currWallIndex]
					}
				}
				
			}
		}
	}
	result := make([]Normal3D, 0, len(neighborNormals))
	for _, normal := range neighborNormals {
		result = append(result, normal)
	}
	
	return result
}