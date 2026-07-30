package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

func sdRoundedRect(px, py, width, height, radius float64) float64 {
	dx := math.Abs(px) - (width - radius)
	dy := math.Abs(py) - (height - radius)
	ax := math.Max(dx, 0)
	ay := math.Max(dy, 0)
	outside := math.Hypot(ax, ay)
	inside := math.Min(math.Max(dx, dy), 0)
	return outside + inside - radius
}

func sdSegment(px, py, ax, ay, bx, by float64) float64 {
	pax, pay := px-ax, py-ay
	bax, bay := bx-ax, by-ay
	b2 := bax*bax + bay*bay
	if b2 == 0 {
		return math.Hypot(pax, pay)
	}
	h := (pax*bax + pay*bay) / b2
	if h < 0 {
		h = 0
	} else if h > 1 {
		h = 1
	}
	dx := pax - bax*h
	dy := pay - bay*h
	return math.Hypot(dx, dy)
}

func sdEllipse(px, py, cx, cy, rx, ry float64) float64 {
	dx := (px - cx) / rx
	dy := (py - cy) / ry
	return math.Hypot(dx, dy) - 1.0
}

func sdCircle(px, py, cx, cy, r float64) float64 {
	return math.Hypot(px-cx, py-cy) - r
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func mixColor(c1, c2 color.RGBA, t float64) color.RGBA {
	t = clamp(t, 0, 1)
	return color.RGBA{
		R: uint8(float64(c1.R)*(1-t) + float64(c2.R)*t),
		G: uint8(float64(c1.G)*(1-t) + float64(c2.G)*t),
		B: uint8(float64(c1.B)*(1-t) + float64(c2.B)*t),
		A: uint8(float64(c1.A)*(1-t) + float64(c2.A)*t),
	}
}

func blendOver(dst, src color.RGBA) color.RGBA {
	sa := float64(src.A) / 255.0
	da := float64(dst.A) / 255.0
	outA := sa + da*(1-sa)
	if outA == 0 {
		return color.RGBA{}
	}
	r := (float64(src.R)*sa + float64(dst.R)*da*(1-sa)) / outA
	g := (float64(src.G)*sa + float64(dst.G)*da*(1-sa)) / outA
	b := (float64(src.B)*sa + float64(dst.B)*da*(1-sa)) / outA
	return color.RGBA{
		R: uint8(clamp(r, 0, 255)),
		G: uint8(clamp(g, 0, 255)),
		B: uint8(clamp(b, 0, 255)),
		A: uint8(clamp(outA*255, 0, 255)),
	}
}

func renderIcon(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	scale := float64(size) / 1024.0

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var rAcc, gAcc, bAcc, aAcc float64
			for sy := 0; sy < 2; sy++ {
				for sx := 0; sx < 2; sx++ {
					px := (float64(x) + (float64(sx)+0.5)/2.0) / scale
					py := (float64(y) + (float64(sy)+0.5)/2.0) / scale

					col := samplePoint(px, py)
					rAcc += float64(col.R)
					gAcc += float64(col.G)
					bAcc += float64(col.B)
					aAcc += float64(col.A)
				}
			}
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(rAcc / 4.0),
				G: uint8(gAcc / 4.0),
				B: uint8(bAcc / 4.0),
				A: uint8(aAcc / 4.0),
			})
		}
	}
	return img
}

func samplePoint(px, py float64) color.RGBA {
	canvas := color.RGBA{0, 0, 0, 0}

	// 1. Squircle App Background
	sqDist := sdRoundedRect(px-512, py-512, 420, 420, 200)

	if sqDist > 2.0 {
		return canvas
	}

	tY := clamp((py-92)/840.0, 0, 1)
	bgTop := color.RGBA{24, 24, 38, 255}
	bgBottom := color.RGBA{10, 10, 16, 255}
	bgColor := mixColor(bgTop, bgBottom, tY)

	distCenter := math.Hypot(px-512, py-512)
	glowFactor := clamp(1.0-distCenter/450.0, 0, 1)
	glowColor := color.RGBA{255, 90, 54, uint8(glowFactor * 45.0)}
	bgColor = blendOver(bgColor, glowColor)

	borderDist := math.Abs(sqDist + 4.0)
	if borderDist < 4.0 {
		borderAlpha := uint8((1.0 - borderDist/4.0) * 120.0)
		borderColor := color.RGBA{255, 90, 54, borderAlpha}
		bgColor = blendOver(bgColor, borderColor)
	}

	bgAlpha := clamp(0.5-sqDist, 0, 1)
	bgColor.A = uint8(float64(bgColor.A) * bgAlpha)
	canvas = blendOver(canvas, bgColor)

	// 2. Crab Legs
	legsColor1 := color.RGBA{220, 60, 30, 255}
	legsColor2 := color.RGBA{255, 90, 54, 255}

	type LegSegment struct {
		x1, y1, x2, y2 float64
	}
	legs := [][]LegSegment{
		{{380, 540, 280, 530}, {280, 530, 230, 580}},
		{{390, 590, 290, 610}, {290, 610, 250, 670}},
		{{410, 640, 320, 680}, {320, 680, 290, 740}},
		{{644, 540, 744, 530}, {744, 530, 794, 580}},
		{{634, 590, 734, 610}, {734, 610, 774, 670}},
		{{614, 640, 704, 680}, {704, 680, 734, 740}},
	}

	minLegDist := 9999.0
	for _, leg := range legs {
		for _, seg := range leg {
			d := sdSegment(px, py, seg.x1, seg.y1, seg.x2, seg.y2)
			if d < minLegDist {
				minLegDist = d
			}
		}
	}
	if minLegDist < 12.0 {
		aa := clamp((12.0-minLegDist)/1.5, 0, 1)
		legCol := mixColor(legsColor1, legsColor2, clamp((py-500)/250.0, 0, 1))
		legCol.A = uint8(float64(legCol.A) * aa)
		canvas = blendOver(canvas, legCol)
	}

	// 3. Crab Shell Body
	shellDist := sdEllipse(px, py, 512, 570, 180, 120) * 120.0
	if shellDist < 1.5 {
		aa := clamp((1.5-shellDist)/1.5, 0, 1)
		shellTop := color.RGBA{255, 110, 70, 255}
		shellBot := color.RGBA{210, 50, 25, 255}
		shellCol := mixColor(shellTop, shellBot, clamp((py-450)/240.0, 0, 1))

		shineDist := sdEllipse(px, py, 512, 515, 130, 45) * 45.0
		if shineDist < 0 {
			shineAlpha := clamp(-shineDist/45.0, 0, 1) * 0.35
			shellCol = blendOver(shellCol, color.RGBA{255, 255, 255, uint8(shineAlpha * 255)})
		}

		shellCol.A = uint8(float64(shellCol.A) * aa)
		canvas = blendOver(canvas, shellCol)
	}

	// Inner Code Terminal Screen
	screenDist := sdRoundedRect(px-512, py-575, 120, 65, 18)
	if screenDist < 1.5 {
		aa := clamp((1.5-screenDist)/1.5, 0, 1)
		screenCol := color.RGBA{16, 16, 24, 255}

		if math.Abs(screenDist+2.0) < 2.0 {
			screenCol = mixColor(screenCol, color.RGBA{255, 90, 54, 180}, 0.6)
		}

		screenCol.A = uint8(float64(screenCol.A) * aa)
		canvas = blendOver(canvas, screenCol)
	}

	// Code Elements: `< / >`
	d1 := sdSegment(px, py, 435, 560, 415, 575)
	d2 := sdSegment(px, py, 415, 575, 435, 590)
	dLeftBracket := math.Min(d1, d2)
	dSlash := sdSegment(px, py, 520, 592, 504, 558)
	d3 := sdSegment(px, py, 589, 560, 609, 575)
	d4 := sdSegment(px, py, 609, 575, 589, 590)
	dRightBracket := math.Min(d3, d4)

	codeDist := math.Min(math.Min(dLeftBracket, dSlash), dRightBracket)
	if codeDist < 4.5 {
		aa := clamp((4.5-codeDist)/1.5, 0, 1)
		codeCol := color.RGBA{255, 110, 75, uint8(aa * 255)}
		canvas = blendOver(canvas, codeCol)
	}

	// 4. Claws (Code Brackets `<` and `>`)
	lc1 := sdSegment(px, py, 370, 500, 260, 400)
	lc2 := sdSegment(px, py, 260, 400, 330, 310)
	lc3 := sdSegment(px, py, 260, 400, 330, 425)
	clawLeftDist := math.Min(math.Min(lc1, lc2), lc3)

	if clawLeftDist < 18.0 {
		aa := clamp((18.0-clawLeftDist)/1.5, 0, 1)
		clawTop := color.RGBA{255, 120, 80, 255}
		clawBot := color.RGBA{220, 55, 30, 255}
		clawCol := mixColor(clawTop, clawBot, clamp((py-300)/200.0, 0, 1))

		tipDist := sdCircle(px, py, 260, 400, 22)
		if tipDist < 0 {
			clawCol = blendOver(clawCol, color.RGBA{255, 200, 160, uint8(clamp(-tipDist/22.0, 0, 1) * 120)})
		}

		clawCol.A = uint8(float64(clawCol.A) * aa)
		canvas = blendOver(canvas, clawCol)
	}

	rc1 := sdSegment(px, py, 654, 500, 764, 400)
	rc2 := sdSegment(px, py, 764, 400, 694, 310)
	rc3 := sdSegment(px, py, 764, 400, 694, 425)
	clawRightDist := math.Min(math.Min(rc1, rc2), rc3)

	if clawRightDist < 18.0 {
		aa := clamp((18.0-clawRightDist)/1.5, 0, 1)
		clawTop := color.RGBA{255, 120, 80, 255}
		clawBot := color.RGBA{220, 55, 30, 255}
		clawCol := mixColor(clawTop, clawBot, clamp((py-300)/200.0, 0, 1))

		tipDist := sdCircle(px, py, 764, 400, 22)
		if tipDist < 0 {
			clawCol = blendOver(clawCol, color.RGBA{255, 200, 160, uint8(clamp(-tipDist/22.0, 0, 1) * 120)})
		}

		clawCol.A = uint8(float64(clawCol.A) * aa)
		canvas = blendOver(canvas, clawCol)
	}

	// 5. Eye Stalks and Tech Eyes
	lsDist := sdSegment(px, py, 440, 475, 425, 410)
	if lsDist < 8.0 {
		aa := clamp((8.0-lsDist)/1.5, 0, 1)
		canvas = blendOver(canvas, color.RGBA{210, 50, 25, uint8(aa * 255)})
	}
	rsDist := sdSegment(px, py, 584, 475, 599, 410)
	if rsDist < 8.0 {
		aa := clamp((8.0-rsDist)/1.5, 0, 1)
		canvas = blendOver(canvas, color.RGBA{210, 50, 25, uint8(aa * 255)})
	}

	lEyeDist := sdCircle(px, py, 425, 400, 22)
	if lEyeDist < 1.5 {
		aa := clamp((1.5-lEyeDist)/1.5, 0, 1)
		eyeCol := color.RGBA{255, 90, 54, uint8(aa * 255)}

		lPupilDist := sdCircle(px, py, 425, 400, 11)
		if lPupilDist < 1.5 {
			pAA := clamp((1.5-lPupilDist)/1.5, 0, 1)
			eyeCol = blendOver(eyeCol, color.RGBA{0, 220, 255, uint8(pAA * 255)})

			lCoreDist := sdCircle(px, py, 422, 397, 4)
			if lCoreDist < 1.5 {
				cAA := clamp((1.5-lCoreDist)/1.5, 0, 1)
				eyeCol = blendOver(eyeCol, color.RGBA{255, 255, 255, uint8(cAA * 255)})
			}
		}
		canvas = blendOver(canvas, eyeCol)
	}

	rEyeDist := sdCircle(px, py, 599, 400, 22)
	if rEyeDist < 1.5 {
		aa := clamp((1.5-rEyeDist)/1.5, 0, 1)
		eyeCol := color.RGBA{255, 90, 54, uint8(aa * 255)}

		rPupilDist := sdCircle(px, py, 599, 400, 11)
		if rPupilDist < 1.5 {
			pAA := clamp((1.5-rPupilDist)/1.5, 0, 1)
			eyeCol = blendOver(eyeCol, color.RGBA{0, 220, 255, uint8(pAA * 255)})

			rCoreDist := sdCircle(px, py, 596, 397, 4)
			if rCoreDist < 1.5 {
				cAA := clamp((1.5-rCoreDist)/1.5, 0, 1)
				eyeCol = blendOver(eyeCol, color.RGBA{255, 255, 255, uint8(cAA * 255)})
			}
		}
		canvas = blendOver(canvas, eyeCol)
	}

	return canvas
}

func main() {
	fmt.Println("Generating official CrabCode app icon...")
	img := renderIcon(1024)

	_ = os.MkdirAll("build", 0755)
	_ = os.MkdirAll("frontend/public", 0755)

	filesToSave := []string{
		"build/appicon.png",
		"frontend/public/appicon.png",
		"frontend/public/favicon.png",
	}

	for _, path := range filesToSave {
		f, err := os.Create(path)
		if err != nil {
			fmt.Printf("Error creating %s: %v\n", path, err)
			continue
		}
		err = png.Encode(f, img)
		f.Close()
		if err != nil {
			fmt.Printf("Error encoding PNG to %s: %v\n", path, err)
		} else {
			fmt.Printf("Successfully generated icon at %s\n", path)
		}
	}
}
