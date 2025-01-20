import numpy as np
from math import sqrt, log10,pi,e
import matplotlib.pyplot as plt
class Raytracing:
    def __init__(self, matrixDimensions, tPos, tPower, tFreq, rFactor, oPos):
        self.step = 0.1
        self.transmitterPos = tPos
        self.transmitterPower = tPower #mW
        self.transmitterFreq = tFreq # GHz
        self.waveLength = 299792458 / tFreq / 10 ** 9;
        self.reflectionFactor = rFactor
        self.obstaclesPos = oPos 
        self.powerMap = np.zeros((matrixDimensions[1]*10+1, matrixDimensions[0]*10+1))
        self.matrix = self.createMatrix(matrixDimensions)
        self.mirroredTransmittersPos = self.createMirroredTransmitters()
    def createMatrix(self, matrixDimensions):
        x = np.linspace(0, matrixDimensions[0], int(matrixDimensions[0]/self.step)+1)
        y = np.linspace(0, matrixDimensions[1], int(matrixDimensions[1]/self.step)+1)

        xv, yv = np.meshgrid(x, y)
        matrix = np.stack((xv, yv), axis=-1)
        return matrix

    def twoVectors(self, A_x, A_y, B_x, B_y, C_x, C_y, D_x, D_y):
        result = (((C_x - A_x)*(B_y - A_y) - (B_x - A_x)*(C_y - A_y)) * ((D_x - A_x) * (B_y - A_y) - (B_x - A_x) * (D_y - A_y)))  #[C_x-A-x, B_x-A_x, vector CA, BA
                                                                                                                                   #C_y-A-y, B_y-A_y]
        if result > 0:
            return -1
        else:
            result2 = (((A_x - C_x)*(D_y - C_y) - (D_x - C_x)*(A_y - C_y)) * ((B_x - C_x) * (D_y - C_y) - (D_x - C_x) * (B_y - C_y)))
            if result2 > 0:
                return -1
            elif result < 0 and result2 < 0:
                return 1
            elif result == 0 and result2 < 0:
                return 0
            elif result < 0 and result2 == 0:
                return 0
            elif A_x < C_x and A_x < D_x and B_x < C_x and B_x < D_x:
                return -1
            elif A_y < C_y and A_y < D_y and B_y < C_y and B_y < D_y:
                return -1
            elif A_x > C_x and A_x > D_x and B_x > C_x and B_x > D_x:
                return -1
            elif A_y > C_y and A_y > D_y and B_y > C_y and B_y > D_y:
                return -1
            else:
                return 0
    def calculateRayTracing(self):
        for i in range(len(self.matrix)):
            for j in range(len(self.matrix[0])):
                H = 0
                receiverPos = self.matrix[i][j]
                checkWall1LineOfSight = self.twoVectors(receiverPos[0], receiverPos[1], self.transmitterPos[0], self.transmitterPos[1], self.obstaclesPos[0][0][0], self.obstaclesPos[0][0][1], self.obstaclesPos[0][1][0], self.obstaclesPos[0][1][1])
                checkWall2LineOfSight = self.twoVectors(receiverPos[0], receiverPos[1], self.transmitterPos[0], self.transmitterPos[1], self.obstaclesPos[1][0][0], self.obstaclesPos[1][0][1], self.obstaclesPos[1][1][0], self.obstaclesPos[1][1][1])
                if checkWall1LineOfSight > 0.1 or checkWall2LineOfSight > 0.1: #checking collision with wall line of sight
                    pass
                else:
                    H += self.calculateTransmitation(receiverPos, self.transmitterPos)  # add to score from line of sight
                checkWall1Reflection = False
                checkWall2Reflection = False
                if self.twoVectors(receiverPos[0], receiverPos[1], self.mirroredTransmittersPos[0][0], self.mirroredTransmittersPos[0][1], self.obstaclesPos[0][0][0], self.obstaclesPos[0][0][1], self.obstaclesPos[0][1][0], self.obstaclesPos[0][1][1]) == 1 and self.twoVectors(receiverPos[0], receiverPos[1], self.mirroredTransmittersPos[0][0], self.mirroredTransmittersPos[0][1], self.obstaclesPos[1][0][0], self.obstaclesPos[1][0][1], self.obstaclesPos[1][1][0], self.obstaclesPos[1][1][1]) == -1:
                        checkWall1Reflection = True
                if self.twoVectors(receiverPos[0], receiverPos[1], self.mirroredTransmittersPos[1][0], self.mirroredTransmittersPos[1][1], self.obstaclesPos[1][0][0], self.obstaclesPos[1][0][1], self.obstaclesPos[1][1][0], self.obstaclesPos[1][1][1]) == 1 and self.twoVectors(receiverPos[0], receiverPos[1], self.mirroredTransmittersPos[1][0], self.mirroredTransmittersPos[1][1], self.obstaclesPos[0][0][0], self.obstaclesPos[0][0][1], self.obstaclesPos[0][1][0], self.obstaclesPos[0][1][1]) == -1:
                        checkWall2Reflection = True

                if checkWall1Reflection: # add to score from 1 wall reflection
                    H += self.calculateTransmitation(receiverPos, self.mirroredTransmittersPos[0])
                if checkWall2Reflection: # add to score from 2 wall reflection
                    H += self.calculateTransmitation(receiverPos, self.mirroredTransmittersPos[1])
                    
                if H == 0:
                    self.powerMap[i][j] = -150
                else:
                    self.powerMap[i][j] = 10*log10(self.transmitterPower) + 20*log10(abs(H))
        

            


    def createMirroredTransmitters(self):
        mirroredTransmittersPos = np.zeros((len(self.obstaclesPos), 2))
        for i in range(len(self.obstaclesPos)):
            if self.obstaclesPos[i][0][0] == self.obstaclesPos[i][1][0]:
                mirroredTransmittersPos[i][1] = self.transmitterPos[1]
                distance = abs(self.obstaclesPos[i][0][0] - self.transmitterPos[0])
                if self.transmitterPos[0] < self.obstaclesPos[i][0][0]:
                    mirroredTransmittersPos[i][0] = self.obstaclesPos[i][0][0] + distance
                else:
                    mirroredTransmittersPos[i][0] = self.obstaclesPos[i][0][0] - distance
            elif self.obstaclesPos[i][0][1] == self.obstaclesPos[i][1][1]:
                mirroredTransmittersPos[i][0] = self.transmitterPos[0]
                distance = abs(self.obstaclesPos[i][0][1] - self.transmitterPos[1])
                if self.transmitterPos[1] < self.obstaclesPos[i][0][1]:
                    mirroredTransmittersPos[i][1] = self.obstaclesPos[i][0][1] + distance
                else:
                    mirroredTransmittersPos[i][1] = self.obstaclesPos[i][0][1] - distance
        return mirroredTransmittersPos
            
    def calculateTransmitation(self, p1, p2, reflectionRef=1):
        r = self.calculateDist(p1, p2)
        H = reflectionRef*self.waveLength/(4*pi*r)*e**(-2j*pi*r/self.waveLength)
        return H
    def calculateDist(self, p1, p2):
        dist = sqrt((p1[0] - p2[0])**2 + (p1[1] - p2[1])**2)
        return dist    
    def displayPowerMap(self):
        plt.figure(figsize=(10, 8))
        plt.imshow(self.powerMap, origin='lower', cmap='jet', extent=[0, self.matrix.shape[1]*0.1, 0, self.matrix.shape[0]*0.1])
        plt.colorbar(label='Power (dBm)')
        plt.title('Power Map')
        plt.xlabel('X Coordinate (m)')
        plt.ylabel('Y Coordinate (m)')
        plt.show()
        
raytracing = Raytracing([16, 28], [12.05, 16.05], 5, 3.6, 0.7, [[[0, 20.05],[10, 20.05]], [[14, 10.05],[14, 15.05]]])
raytracing.calculateRayTracing()
raytracing.displayPowerMap()