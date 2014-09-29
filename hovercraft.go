package main

import (
	"github.com/go-gl/glow/gl/2.1/gl"
	"github.com/go-gl/mathgl/mgl64"
	glfw "github.com/shurcooL/glfw3"
)

type Hovercraft struct {
	x float64
	y float64
	z float64

	r float64
}

func (this *Hovercraft) Render() {
	gl.PushMatrix()
	defer gl.PopMatrix()

	gl.Translated(float64(this.x), float64(this.y), float64(this.z))
	gl.Rotated(float64(this.r), 0, 0, -1)

	gl.Begin(gl.TRIANGLES)
	{
		const size = 1
		gl.Color3f(0, 1, 0)
		gl.Vertex3i(0, 0, 0)
		gl.Vertex3i(0, +size, 3*size)
		gl.Vertex3i(0, -size, 3*size)
		gl.Color3f(1, 0, 0)
		gl.Vertex3i(0, 0, 0)
		gl.Vertex3i(0, +size, -3*size)
		gl.Vertex3i(0, -size, -3*size)
	}
	gl.End()
}

func (this *Hovercraft) Input(window *glfw.Window) {
	if (mustAction(window.GetKey(glfw.KeyLeft)) != glfw.Release) && !(mustAction(window.GetKey(glfw.KeyRight)) != glfw.Release) {
		this.r -= 3
	} else if (mustAction(window.GetKey(glfw.KeyRight)) != glfw.Release) && !(mustAction(window.GetKey(glfw.KeyLeft)) != glfw.Release) {
		this.r += 3
	}

	var direction mgl64.Vec2
	if (mustAction(window.GetKey(glfw.KeyA)) != glfw.Release) && !(mustAction(window.GetKey(glfw.KeyD)) != glfw.Release) {
		direction[1] = +1
	} else if (mustAction(window.GetKey(glfw.KeyD)) != glfw.Release) && !(mustAction(window.GetKey(glfw.KeyA)) != glfw.Release) {
		direction[1] = -1
	}
	if (mustAction(window.GetKey(glfw.KeyW)) != glfw.Release) && !(mustAction(window.GetKey(glfw.KeyS)) != glfw.Release) {
		direction[0] = +1
	} else if (mustAction(window.GetKey(glfw.KeyS)) != glfw.Release) && !(mustAction(window.GetKey(glfw.KeyW)) != glfw.Release) {
		direction[0] = -1
	}
	if (mustAction(window.GetKey(glfw.KeyQ)) != glfw.Release) && !(mustAction(window.GetKey(glfw.KeyE)) != glfw.Release) {
		this.z -= 1
	} else if (mustAction(window.GetKey(glfw.KeyE)) != glfw.Release) && !(mustAction(window.GetKey(glfw.KeyQ)) != glfw.Release) {
		this.z += 1
	}

	// Physics update.
	if direction.Len() != 0 {
		rotM := mgl64.Rotate2D(mgl64.DegToRad(-this.r))
		direction = rotM.Mul2x1(direction)

		direction = direction.Normalize().Mul(1)

		if mustAction(window.GetKey(glfw.KeyLeftShift)) != glfw.Release || mustAction(window.GetKey(glfw.KeyRightShift)) != glfw.Release {
			direction = direction.Mul(0.001)
		} else if mustAction(window.GetKey(glfw.KeySpace)) != glfw.Release {
			direction = direction.Mul(5)
		}

		this.x += direction[0]
		this.y += direction[1]
	}
}

// Update physics.
func (this *Hovercraft) Physics() {
	// TEST: Check terrain height calculations.
	{
		this.z = track.getHeightAt(this.x, this.y)
	}
}