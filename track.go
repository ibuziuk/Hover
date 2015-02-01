package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"

	glfw "github.com/shurcooL/goglfw"
	"github.com/shurcooL/webgl"
)

var track *Track

const TRIGROUP_NUM_BITS_USED = 510
const TRIGROUP_NUM_DWORDS = (TRIGROUP_NUM_BITS_USED + 2) / 32
const TRIGROUP_WIDTHSHIFT = 4
const TERR_HEIGHT_SCALE = 1.0 / 32

type TerrTypeNode struct {
	Type       uint8
	_          uint8
	NextStartX uint16
	Next       uint16
}

type NavCoord struct {
	X, Z             uint16
	DistToStartCoord uint16 // Decider at forks, and determines racers' rank/place.
	Next             uint16
	Alt              uint16
}

type NavCoordLookupNode struct {
	NavCoord   uint16
	NextStartX uint16
	Next       uint16
}

type TerrCoord struct {
	Height         uint16
	LightIntensity uint8
}

type TriGroup struct {
	Data [TRIGROUP_NUM_DWORDS]uint32
}

type TrackFileHeader struct {
	SunlightDirection, SunlightPitch float32
	RacerStartPositions              [8][3]float32
	NumTerrTypes                     uint16
	NumTerrTypeNodes                 uint16
	NumNavCoords                     uint16
	NumNavCoordLookupNodes           uint16
	Width, Depth                     uint16
}

type Track struct {
	TrackFileHeader
	NumTerrCoords  uint32
	TriGroupsWidth uint32
	TriGroupsDepth uint32
	NumTriGroups   uint32

	TerrTypeTextureFilenames []string

	TerrTypeRuns  []TerrTypeNode
	TerrTypeNodes []TerrTypeNode

	NavCoords           []NavCoord
	NavCoordLookupRuns  []NavCoordLookupNode
	NavCoordLookupNodes []NavCoordLookupNode

	TerrCoords []TerrCoord
	TriGroups  []TriGroup

	vertexVbo   *webgl.Buffer
	colorVbo    *webgl.Buffer
	terrTypeVbo *webgl.Buffer
}

func newTrack(path string) *Track {
	started := time.Now()

	file, err := glfw.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var track Track

	binary.Read(file, binary.LittleEndian, &track.TrackFileHeader)

	// Stuff derived from header info.
	track.NumTerrCoords = uint32(track.Width) * uint32(track.Depth)
	track.TriGroupsWidth = (uint32(track.Width) - 1) >> TRIGROUP_WIDTHSHIFT
	track.TriGroupsDepth = (uint32(track.Depth) - 1) >> TRIGROUP_WIDTHSHIFT
	track.NumTriGroups = track.TriGroupsWidth * track.TriGroupsDepth

	track.TerrTypeTextureFilenames = make([]string, track.NumTerrTypes)
	for i := uint16(0); i < track.NumTerrTypes; i++ {
		var terrTypeTextureFilename [32]byte
		binary.Read(file, binary.LittleEndian, &terrTypeTextureFilename)
		track.TerrTypeTextureFilenames[i] = cStringToGoString(terrTypeTextureFilename[:])
	}

	track.TerrTypeRuns = make([]TerrTypeNode, track.Depth)
	binary.Read(file, binary.LittleEndian, &track.TerrTypeRuns)

	track.TerrTypeNodes = make([]TerrTypeNode, track.NumTerrTypeNodes)
	binary.Read(file, binary.LittleEndian, &track.TerrTypeNodes)

	track.NavCoords = make([]NavCoord, track.NumNavCoords)
	binary.Read(file, binary.LittleEndian, &track.NavCoords)

	track.NavCoordLookupRuns = make([]NavCoordLookupNode, track.Depth)
	binary.Read(file, binary.LittleEndian, &track.NavCoordLookupRuns)

	track.NavCoordLookupNodes = make([]NavCoordLookupNode, track.NumNavCoordLookupNodes)
	binary.Read(file, binary.LittleEndian, &track.NavCoordLookupNodes)

	track.TerrCoords = make([]TerrCoord, track.NumTerrCoords)
	binary.Read(file, binary.LittleEndian, &track.TerrCoords)

	track.TriGroups = make([]TriGroup, track.NumTriGroups)
	binary.Read(file, binary.LittleEndian, &track.TriGroups)

	fileOffset, err := file.Seek(0, os.SEEK_CUR)
	if err != nil {
		panic(err)
	}
	fileSize, err := file.Seek(0, os.SEEK_END)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Read %v of %v bytes.\n", fileOffset, fileSize)

	{
		rowCount := int(track.Depth) - 1
		rowLength := int(track.Width)

		terrTypeMap := make([]uint8, int(track.Width)*int(track.Depth))
		for y := 0; y < int(track.Depth); y++ {
			pCurrNode := &track.TerrTypeRuns[y]

			for x := 0; x < int(track.Width); x++ {
				if x >= int(pCurrNode.NextStartX) {
					pCurrNode = &track.TerrTypeNodes[pCurrNode.Next]
				}
				terrTypeMap[y*int(track.Width)+x] = pCurrNode.Type
			}
		}

		vertexData := make([]float32, 3*2*rowLength*rowCount)
		colorData := make([]uint8, 3*2*rowLength*rowCount)
		terrTypeData := make([]float32, 2*rowLength*rowCount)

		var index int
		for y := 1; y < int(track.Depth); y++ {
			for x := 0; x < int(track.Width); x++ {
				for i := 0; i < 2; i++ {
					yy := y - i

					terrCoord := &track.TerrCoords[yy*int(track.Width)+x]
					height := float64(terrCoord.Height) * TERR_HEIGHT_SCALE
					lightIntensity := uint8(terrCoord.LightIntensity)

					vertexData[3*index+0], vertexData[3*index+1], vertexData[3*index+2] = float32(x), float32(yy), float32(height)
					colorData[3*index+0], colorData[3*index+1], colorData[3*index+2] = lightIntensity, lightIntensity, lightIntensity
					if terrTypeMap[yy*int(track.Width)+x] == 0 {
						terrTypeData[index] = 0
					} else {
						terrTypeData[index] = 1
					}
					index++
				}
			}
		}

		track.vertexVbo = createVbo3Float(vertexData)
		track.colorVbo = createVbo3Ubyte(colorData)
		track.terrTypeVbo = createVbo3Float(terrTypeData)
	}

	fmt.Println("Done loading track in:", time.Since(started))

	return &track
}

func (track *Track) getHeightAt(x, y float64) float64 {
	// TODO: Interpolate between 4 points.
	return track.getHeightAtPoint(uint64(x), uint64(y))
}

func (track *Track) getHeightAtPoint(x, y uint64) float64 {
	if x > uint64(track.Width)-1 {
		x = uint64(track.Width) - 1
	}
	if y > uint64(track.Depth)-1 {
		y = uint64(track.Depth) - 1
	}

	terrCoord := track.TerrCoords[y*uint64(track.Width)+x]
	height := float64(terrCoord.Height) * TERR_HEIGHT_SCALE
	return height
}

func (track *Track) Render() {
	if wireframe {
		//gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
	}

	gl.UseProgram(program)
	{
		rowCount := int(track.Depth) - 1
		rowLength := int(track.Width)

		gl.BindBuffer(gl.ARRAY_BUFFER, track.vertexVbo)
		vertexPositionAttribute := gl.GetAttribLocation(program, "aVertexPosition")
		gl.EnableVertexAttribArray(vertexPositionAttribute)
		gl.VertexAttribPointer(vertexPositionAttribute, 3, gl.FLOAT, false, 0, 0)

		gl.BindBuffer(gl.ARRAY_BUFFER, track.colorVbo)
		vertexColorAttribute := gl.GetAttribLocation(program, "aVertexColor")
		gl.EnableVertexAttribArray(vertexColorAttribute)
		gl.VertexAttribPointer(vertexColorAttribute, 3, gl.UNSIGNED_BYTE, true, 0, 0)

		gl.BindBuffer(gl.ARRAY_BUFFER, track.terrTypeVbo)
		vertexTerrTypeAttribute := gl.GetAttribLocation(program, "aVertexTerrType")
		gl.EnableVertexAttribArray(vertexTerrTypeAttribute)
		gl.VertexAttribPointer(vertexTerrTypeAttribute, 1, gl.FLOAT, false, 0, 0)

		for row := 0; row < rowCount; row++ {
			gl.DrawArrays(gl.TRIANGLE_STRIP, row*2*rowLength, 2*rowLength)
		}
	}
	gl.UseProgram(nil)

	if wireframe {
		//gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	}
}

// ---

func createVbo3Float(vertices []float32) *webgl.Buffer {
	vbo := gl.CreateBuffer()
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, vertices, gl.STATIC_DRAW)
	return vbo
}

func createVbo3Ubyte(vertices []uint8) *webgl.Buffer {
	vbo := gl.CreateBuffer()
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, vertices, gl.STATIC_DRAW)
	return vbo
}

func cStringToGoString(cString []byte) string {
	n := 0
	for i, b := range cString {
		if b == 0 {
			break
		}
		n = i + 1
	}
	return string(cString[:n])
}
