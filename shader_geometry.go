package main

import (
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/light"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
)

type ShaderGeometry struct {
	ctx           *Context
	plane         *graphic.Mesh
	box           *graphic.Mesh
	sphere        *graphic.Mesh
	showWireframe int32
	showVnormal   int32
	showFnormal   int32
	rotate        bool
}

func init() {
	TestMap["shader.geometry"] = &ShaderGeometry{}
}

func (t *ShaderGeometry) Initialize(ctx *Context) {

	// Add help label
	const help = `Wireframe, vertex and face normals generated by Geometry Shader`
	label1 := gui.NewLabel(help)
	label1.SetFontSize(16)
	label1.SetPosition(10, 10)
	ctx.Gui.Add(label1)

	// Adds directional front light
	dir1 := light.NewDirectional(math32.NewColor(1, 1, 1), 0.6)
	dir1.SetPosition(0, 0, 100)
	ctx.Scene.Add(dir1)

	// Add axis helper
	axis := graphic.NewAxisHelper(1)
	ctx.Scene.Add(axis)

	// Registers shaders and program
	ctx.Renderer.AddShader("shaderGSDemoVertex", sourceGSDemoVertex)
	ctx.Renderer.AddShader("shaderGSDemoGeometry", sourceGSDemoGeometry)
	ctx.Renderer.AddShader("shaderGSDemoFrag", sourceGSDemoFrag)
	ctx.Renderer.AddProgram("progGSDemo", "shaderGSDemoVertex", "shaderGSDemoFrag", "shaderGSDemoGeometry")

	// Creates shared custom material to show normals
	mat := newNormalsMaterial()

	// Adds rectangular plane
	planeGeom := geometry.NewPlane(1, 1, 1, 1)
	mat.Incref()
	t.plane = graphic.NewMesh(planeGeom, mat)
	t.plane.SetPosition(-2.2, 0, 0)
	ctx.Scene.Add(t.plane)

	// Adds box
	boxGeom := geometry.NewBox(1, 1, 1, 1, 1, 1)
	mat.Incref()
	t.box = graphic.NewMesh(boxGeom, mat)
	t.box.SetPosition(0, 0, 0)
	ctx.Scene.Add(t.box)

	// Adds sphere
	sphereGeom := geometry.NewSphere(0.8, 4, 4, 0, math32.Pi*2, 0, math32.Pi)
	mat.Incref()
	t.sphere = graphic.NewMesh(sphereGeom, mat)
	t.sphere.SetPosition(2.2, 0, 0)
	ctx.Scene.Add(t.sphere)

	// Add controls
	if ctx.Control == nil {
		return
	}
	t.showWireframe = 1
	t.showVnormal = 1
	t.showFnormal = 1
	t.rotate = true
	g1 := ctx.Control.AddGroup("Show")
	cb0 := g1.AddCheckBox("Rotate").SetValue(true)
	cb0.Subscribe(gui.OnChange, func(evname string, ev interface{}) {
		t.rotate = !t.rotate
	})
	cb1 := g1.AddCheckBox("Wireframe").SetValue(true)
	cb1.Subscribe(gui.OnChange, func(evname string, ev interface{}) {
		if t.showWireframe == 0 {
			t.showWireframe = 1
		} else {
			t.showWireframe = 0
		}
		mat.ShowWireframe.Set(t.showWireframe)
	})
	cb2 := g1.AddCheckBox("Vertex normals").SetValue(true)
	cb2.Subscribe(gui.OnChange, func(evname string, ev interface{}) {
		if t.showVnormal == 0 {
			t.showVnormal = 1
		} else {
			t.showVnormal = 0
		}
		mat.ShowVnormal.Set(t.showVnormal)
	})
	cb3 := g1.AddCheckBox("Face normals").SetValue(true)
	cb3.Subscribe(gui.OnChange, func(evname string, ev interface{}) {
		if t.showFnormal == 0 {
			t.showFnormal = 1
		} else {
			t.showFnormal = 0
		}
		mat.ShowFnormal.Set(t.showFnormal)
	})
}

func (t *ShaderGeometry) Render(ctx *Context) {

	if t.rotate {
		t.plane.AddRotationX(0.01)
		t.box.AddRotationY(0.01)
		t.sphere.AddRotationZ(0.005)
	}
}

//
// Normals Custom material
//
type NormalsMaterial struct {
	material.Material // Embedded material
	ShowWireframe     gls.Uniform1i
	ShowVnormal       gls.Uniform1i
	ShowFnormal       gls.Uniform1i
}

func newNormalsMaterial() *NormalsMaterial {

	m := new(NormalsMaterial)
	m.Material.Init()
	m.SetShader("progGSDemo")

	// Creates uniforms
	m.ShowWireframe.Init("ShowWireframe")
	m.ShowVnormal.Init("ShowVnormal")
	m.ShowFnormal.Init("ShowFnormal")

	// Set uniform's initial values
	m.ShowWireframe.Set(1)
	m.ShowVnormal.Set(1)
	m.ShowFnormal.Set(1)
	return m
}

func (m *NormalsMaterial) RenderSetup(gs *gls.GLS) {

	m.Material.RenderSetup(gs)
	m.ShowWireframe.Transfer(gs)
	m.ShowVnormal.Transfer(gs)
	m.ShowFnormal.Transfer(gs)
}

//
// Vertex Shader
// This is pass-through vertex shader which
// sends its input directly to the geometry shader
// without any processing.
//
const sourceGSDemoVertex = `
#version {{.Version}}

{{template "attributes" .}}

// Outputs for geometry shader
out vec3 vnormal;

void main() {

	gl_Position = vec4(VertexPosition, 1.0);
  	vnormal = VertexNormal;
}

`

//
// Geometry Shader
// This geometry shader receives triangles vertices
// from the vertex shader and generates lines for
// wireframe and/or vertex normals and/or face normals.
//
const sourceGSDemoGeometry = `
#version {{.Version}}

layout (triangles) in;
layout (line_strip, max_vertices = 12) out;

// Model uniforms
uniform mat4 MVP;

// Inputs from Vertex Shader
in vec3 vnormal[];

// Inputs uniforms
uniform int ShowWireframe;
uniform int ShowVnormal;
uniform int ShowFnormal;

// Colors
const vec4 colorWire    = vec4(1, 1, 0, 1);
const vec4 colorVnormal = vec4(1, 0, 0, 1);
const vec4 colorFnormal = vec4(0, 0, 1, 1);

// Output color to fragment shader
out vec4 vertex_color;

void main() {

	// Emits triangle's vertices as lines to show wireframe
	if (ShowWireframe != 0) {
		for (int n = 0; n < gl_in.length(); n++) {
			// Vertex position
			gl_Position = MVP * gl_in[n].gl_Position;
			vertex_color = colorWire;
			EmitVertex();
		}
		// Emit first triangle vertex to close the last line strip.
		gl_Position = MVP * gl_in[0].gl_Position;
		vertex_color = colorWire;
		EmitVertex();
		EndPrimitive();
	}

	// Emits lines representing the vertices normals
	if (ShowVnormal != 0) {
		for (int i = 0; i < gl_in.length(); i++) {

			vec3 position = gl_in[i].gl_Position.xyz;
			vec3 normal = vnormal[i];
			
			gl_Position = MVP * vec4(position, 1.0);
			vertex_color = colorVnormal;
			EmitVertex();
			
			gl_Position = MVP * vec4(position + normal * 0.5, 1.0);
			vertex_color = colorVnormal;
			EmitVertex();
			
			EndPrimitive();
		}
	}

	// Emits one line representing the face normal
	if (ShowFnormal != 0) {
		vec3 p0 = gl_in[0].gl_Position.xyz;
		vec3 p1 = gl_in[1].gl_Position.xyz;
		vec3 p2 = gl_in[2].gl_Position.xyz;
	  
		vec3 v0 = p0 - p1;
		vec3 v1 = p2 - p1;
		vec3 faceN = normalize(cross(v1, v0));

		// Center of the triangle
		vec3 center = (p0 + p1 + p2) / 3.0;
	  
		gl_Position = MVP * vec4(center, 1.0);
		vertex_color = colorFnormal;
		EmitVertex();
	  
		gl_Position = MVP * vec4(center + faceN * 0.5, 1.0);
		vertex_color = colorFnormal;
		EmitVertex();
		EndPrimitive();
	}
}

`

//
// Fragment Shader template
//
const sourceGSDemoFrag = `
#version {{.Version}}

in vec4 vertex_color;
out vec4 Out_Color;

void main() {
	Out_Color = vertex_color;
}

`
