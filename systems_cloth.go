package katsu2d

import (
	"math"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	maxClothStepDt                = 1.0 / 30.0
	clothForceScale               = 12.0
	clothWindForceScale           = 2.2
	clothCullPad                  = 64.0
	clothMinShade                 = 0.42
	clothMaxShade                 = 1.12
	clothStructuralStiffnessScale = 1.0
	clothHorizMinRatio            = 0.25
	clothHorizCompressScale       = 0.15
	clothVertMinRatio             = 0.65
	clothVertCompressScale        = 0.25
	clothShearStiffnessScale      = 0.65
	clothBendStiffnessScale       = 0.30
)

type clothActiveGust struct {
	gust               StrongWindGust
	pos                Vector
	bounds             Rectangle
	strengthMultiplier float64
}

type clothLocalGust struct {
	pos                Vector
	dir                Vector
	halfWidth          float64
	halfLength         float64
	strength           float64
	strengthMultiplier float64
}

// ClothSystem simulates 2D Verlet cloth physics, ambient traveling wind flutter,
// radial force interactions, strong wind gusts, and fold shading for cloth entities.
type ClothSystem struct {
	clothFilter      *teishoku.Filter4[TransformComponent, SpriteComponent, MeshComponent, ClothComponent]
	camHandle        teishoku.ResourceHandle[ActiveCamera]
	scratchTransform *Transform
	forceSources     []ForceSource
	windGusts        []StrongWindGust
	activeGusts      []clothActiveGust
	localForcesBuf   []ForceSource
	localGustsBuf    []clothLocalGust
	initialized      bool
}

// NewClothSystem creates and returns a new ClothSystem.
func NewClothSystem() *ClothSystem {
	return &ClothSystem{
		scratchTransform: T(),
	}
}

// Initialize caches the ECS filter and ActiveCamera resource handle required by ClothSystem.
func (s *ClothSystem) Initialize(w *teishoku.World) {
	if s.initialized || w == nil {
		return
	}
	s.clothFilter = teishoku.NewFilter4[TransformComponent, SpriteComponent, MeshComponent, ClothComponent](w)
	s.camHandle = teishoku.UseResource[ActiveCamera](w.Resources())
	if s.scratchTransform == nil {
		s.scratchTransform = T()
	}
	s.initialized = true
}

// SetForceSources replaces the active world-space radial force sources while reusing slice capacity.
func (s *ClothSystem) SetForceSources(sources ...ForceSource) {
	s.forceSources = append(s.forceSources[:0], sources...)
}

// AddWindGust activates and queues a traveling world-space wind gust affecting cloths in its path.
func (s *ClothSystem) AddWindGust(gust StrongWindGust) {
	if gust.Direction.IsZero() {
		diff := gust.EndPos.Sub(gust.StartPos)
		if !diff.IsZero() {
			gust.Direction = diff.Normalize()
		}
	} else {
		gust.Direction = gust.Direction.Normalize()
	}
	gust.Active = true
	s.windGusts = append(s.windGusts, gust)
}

// Update advances all visible cloth simulations by dt seconds (clamped to 1/30s for Verlet stability).
func (s *ClothSystem) Update(w *teishoku.World, dt float64) {
	if w == nil || dt <= 0 {
		return
	}
	if !s.initialized {
		s.Initialize(w)
	}

	dt = min(dt, maxClothStepDt)
	s.advanceWindGusts(dt)

	cam := s.camHandle.Get()
	if cam == nil {
		if ok, _ := teishoku.HasResource[ActiveCamera](w.Resources()); ok {
			s.camHandle = teishoku.UseResource[ActiveCamera](w.Resources())
			cam = s.camHandle.Get()
		}
	}
	hasCam := cam != nil && cam.HasCamera

	chunks := s.clothFilter.Chunks()
	for chunks.Next() {
		transforms, sprites, meshes, cloths := chunks.Get()
		for i := range cloths {
			tr := &transforms[i]
			spr := &sprites[i]
			mesh := &meshes[i]
			cloth := &cloths[i]

			if spr.Width <= 0 || spr.Height <= 0 {
				continue
			}

			clothBounds := computeClothCullBounds(tr, spr.Width, spr.Height)
			if hasCam && !cam.IsVisible(clothBounds) {
				continue
			}

			cols := cloth.Cols
			if cols <= 0 {
				cols = defaultClothCols
			}
			rows := cloth.Rows
			if rows <= 0 {
				rows = defaultClothRows
			}

			wasDirty := mesh.IsDirty
			if mesh.Cols != cols || mesh.Rows != rows {
				mesh.Cols = cols
				mesh.Rows = rows
				mesh.MeshType = MeshTypeGrid
				mesh.IsDirty = true
			}
			GenerateMesh(mesh, spr)

			vertCount := len(mesh.BaseVertices)
			if vertCount == 0 {
				continue
			}

			if !cloth.initialized || wasDirty || len(cloth.posX) != vertCount || cloth.cachedW != spr.Width || cloth.cachedH != spr.Height {
				initClothState(cloth, mesh, cols, rows, spr.Width, spr.Height)
			}

			if len(mesh.Vertices) != vertCount {
				if cap(mesh.Vertices) < vertCount {
					mesh.Vertices = make([]ebiten.Vertex, vertCount)
				} else {
					mesh.Vertices = mesh.Vertices[:vertCount]
				}
				copy(mesh.Vertices, mesh.BaseVertices)
			}

			s.prepareLocalInteractions(tr, clothBounds)
			stepClothVerlet(cloth, cols, rows, dt, cloth.ExternalForces, s.localForcesBuf, s.localGustsBuf)
			solveClothConstraints(cloth, cols, rows, s.localGustsBuf)
			syncClothMesh(cloth, mesh, cols, rows)
		}
	}
}

func (s *ClothSystem) advanceWindGusts(dt float64) {
	s.activeGusts = s.activeGusts[:0]
	n := 0
	gusts := s.windGusts
	for i := range gusts {
		gust := &gusts[i]
		if !gust.Active {
			continue
		}
		gust.ElapsedTime += dt
		if gust.Duration <= 0 || gust.ElapsedTime >= gust.Duration {
			gust.Active = false
			continue
		}

		strengthMultiplier := 1.0
		if gust.FadeInDuration > 0 && gust.ElapsedTime < gust.FadeInDuration {
			strengthMultiplier = gust.ElapsedTime / gust.FadeInDuration
		} else if gust.FadeOutDuration > 0 && gust.ElapsedTime > gust.Duration-gust.FadeOutDuration {
			strengthMultiplier = (gust.Duration - gust.ElapsedTime) / gust.FadeOutDuration
		}
		strengthMultiplier = Clamp(strengthMultiplier, 0.0, 1.0)

		progress := gust.ElapsedTime / gust.Duration
		currentPos := gust.StartPos.Add(gust.EndPos.Sub(gust.StartPos).ScaleF(progress))

		halfLength := gust.Length * 0.5
		halfWidth := gust.Width * 0.5
		if halfLength > 0 && halfWidth > 0 {
			perp := Vector{X: -gust.Direction.Y, Y: gust.Direction.X}.Normalize()
			alongHalf := gust.Direction.ScaleF(halfLength)
			perpHalf := perp.ScaleF(halfWidth)
			c1 := currentGustPosCorner(currentPos, alongHalf, perpHalf, 1, 1)
			c2 := currentGustPosCorner(currentPos, alongHalf, perpHalf, 1, -1)
			c3 := currentGustPosCorner(currentPos, alongHalf, perpHalf, -1, 1)
			c4 := currentGustPosCorner(currentPos, alongHalf, perpHalf, -1, -1)

			s.activeGusts = append(s.activeGusts, clothActiveGust{
				gust: *gust,
				pos:  currentPos,
				bounds: Rectangle{
					Min: Vector{
						X: min(c1.X, c2.X, c3.X, c4.X),
						Y: min(c1.Y, c2.Y, c3.Y, c4.Y),
					},
					Max: Vector{
						X: max(c1.X, c2.X, c3.X, c4.X),
						Y: max(c1.Y, c2.Y, c3.Y, c4.Y),
					},
				},
				strengthMultiplier: strengthMultiplier,
			})
		}

		gusts[n] = *gust
		n++
	}
	s.windGusts = gusts[:n]
}

func currentGustPosCorner(center, along, perp Vector, sAlong, sPerp float64) Vector {
	return Vector{
		X: center.X + along.X*sAlong + perp.X*sPerp,
		Y: center.Y + along.Y*sAlong + perp.Y*sPerp,
	}
}

func computeClothCullBounds(tr *TransformComponent, w, h int) Rectangle {
	sx, sy := tr.Scale.X, tr.Scale.Y
	ox, oy := tr.Offset.X*sx, tr.Offset.Y*sy
	px, py := tr.Position.X-tr.Origin.X, tr.Position.Y-tr.Origin.Y
	fw, fh := float64(w)*sx, float64(h)*sy

	var minX, maxX, minY, maxY float64
	if tr.Rotation == 0 {
		x0 := -ox + px
		x1 := fw - ox + px
		y0 := -oy + py
		y1 := fh - oy + py
		minX, maxX = min(x0, x1), max(x0, x1)
		minY, maxY = min(y0, y1), max(y0, y1)
	} else {
		cos, sin := math.Cos(tr.Rotation), math.Sin(tr.Rotation)
		lx0, lx1 := -ox, fw-ox
		ly0, ly1 := -oy, fh-oy

		tx0, ty0 := lx0*cos-ly0*sin+px, lx0*sin+ly0*cos+py
		tx1, ty1 := lx1*cos-ly0*sin+px, lx1*sin+ly0*cos+py
		tx2, ty2 := lx0*cos-ly1*sin+px, lx0*sin+ly1*cos+py
		tx3, ty3 := lx1*cos-ly1*sin+px, lx1*sin+ly1*cos+py

		minX = min(tx0, tx1, tx2, tx3)
		maxX = max(tx0, tx1, tx2, tx3)
		minY = min(ty0, ty1, ty2, ty3)
		maxY = max(ty0, ty1, ty2, ty3)
	}

	margin := math.Max(maxX-minX, maxY-minY) + clothCullPad
	return Rectangle{
		Min: Vector{X: minX - margin, Y: minY - margin},
		Max: Vector{X: maxX + margin, Y: maxY + margin},
	}
}

func (s *ClothSystem) prepareLocalInteractions(tr *TransformComponent, clothBounds Rectangle) {
	s.localForcesBuf = s.localForcesBuf[:0]
	s.localGustsBuf = s.localGustsBuf[:0]

	if len(s.forceSources) == 0 && len(s.activeGusts) == 0 {
		return
	}

	simpleTransform := tr.Rotation == 0 &&
		tr.Scale.X == 1 && tr.Scale.Y == 1 &&
		tr.Offset.X == 0 && tr.Offset.Y == 0 &&
		tr.Origin.X == 0 && tr.Origin.Y == 0

	var inv Matrix
	invScale := 1.0
	if !simpleTransform {
		if math.Abs(tr.Scale.X) < 1e-6 || math.Abs(tr.Scale.Y) < 1e-6 {
			return
		}
		s.scratchTransform.SetFromComponent(tr)
		inv = s.scratchTransform.Matrix()
		det := inv.Element(0, 0)*inv.Element(1, 1) - inv.Element(0, 1)*inv.Element(1, 0)
		if math.Abs(det) < 1e-12 {
			return
		}
		inv.Invert()
		avgScale := (math.Abs(tr.Scale.X) + math.Abs(tr.Scale.Y)) * 0.5
		if avgScale > 1e-6 {
			invScale = 1.0 / avgScale
		}
	}

	for i := range s.forceSources {
		fs := s.forceSources[i]
		if fs.Radius <= 0 {
			continue
		}
		if simpleTransform {
			fs.Position.X -= tr.Position.X
			fs.Position.Y -= tr.Position.Y
		} else {
			lx, ly := inv.Apply(fs.Position.X, fs.Position.Y)
			fs.Position = Vector{X: lx, Y: ly}
			fs.Radius *= invScale
		}
		s.localForcesBuf = append(s.localForcesBuf, fs)
	}

	for i := range s.activeGusts {
		ag := &s.activeGusts[i]
		if !clothBounds.Intersects(ag.bounds) {
			continue
		}
		localPos := Vector{
			X: ag.pos.X - tr.Position.X,
			Y: ag.pos.Y - tr.Position.Y,
		}
		localDir := ag.gust.Direction
		halfWidth := ag.gust.Width * 0.5
		halfLength := ag.gust.Length * 0.5
		if !simpleTransform {
			lx, ly := inv.Apply(ag.pos.X, ag.pos.Y)
			localPos = Vector{X: lx, Y: ly}
			dx := inv.Element(0, 0)*ag.gust.Direction.X + inv.Element(0, 1)*ag.gust.Direction.Y
			dy := inv.Element(1, 0)*ag.gust.Direction.X + inv.Element(1, 1)*ag.gust.Direction.Y
			localDir = Vector{X: dx, Y: dy}.Normalize()
			halfWidth *= invScale
			halfLength *= invScale
		}
		s.localGustsBuf = append(s.localGustsBuf, clothLocalGust{
			pos:                localPos,
			dir:                localDir,
			halfWidth:          halfWidth,
			halfLength:         halfLength,
			strength:           ag.gust.Strength,
			strengthMultiplier: ag.strengthMultiplier,
		})
	}
}

func initClothState(c *ClothComponent, mesh *MeshComponent, cols, rows, width, height int) {
	vertCount := len(mesh.BaseVertices)
	c.posX = ensureFloat64Slice(c.posX, vertCount)
	c.posY = ensureFloat64Slice(c.posY, vertCount)
	c.prevX = ensureFloat64Slice(c.prevX, vertCount)
	c.prevY = ensureFloat64Slice(c.prevY, vertCount)
	c.restX = ensureFloat64Slice(c.restX, vertCount)
	c.restY = ensureFloat64Slice(c.restY, vertCount)
	c.pinned = ensureBoolSlice(c.pinned, vertCount)
	c.pinWeight = ensureFloat64Slice(c.pinWeight, vertCount)
	c.backFaceAccum = ensureFloat32Slice(c.backFaceAccum, vertCount)
	clear(c.backFaceAccum)
	c.quadOrder = ensureClothQuadDepthSlice(c.quadOrder, cols*rows)

	stride := cols + 1
	hasPinned := false
	for r := 0; r <= rows; r++ {
		rowOff := r * stride
		for col := 0; col <= cols; col++ {
			idx := rowOff + col
			x := float64(mesh.BaseVertices[idx].DstX)
			y := float64(mesh.BaseVertices[idx].DstY)
			c.restX[idx] = x
			c.restY[idx] = y
			c.posX[idx] = x
			c.posY[idx] = y
			c.prevX[idx] = x
			c.prevY[idx] = y

			p := isClothVertexPinned(c, col, r, cols, rows)
			c.pinned[idx] = p
			if p {
				hasPinned = true
			}
		}
	}

	if !hasPinned {
		for i := range vertCount {
			c.pinWeight[i] = 1.0
		}
	} else {
		maxDist := 0.0
		for i := range vertCount {
			if c.pinned[i] {
				c.pinWeight[i] = 0.0
				continue
			}
			minDistSq := math.MaxFloat64
			xi, yi := c.restX[i], c.restY[i]
			for j := range vertCount {
				if !c.pinned[j] {
					continue
				}
				dx := xi - c.restX[j]
				dy := yi - c.restY[j]
				d2 := dx*dx + dy*dy
				if d2 < minDistSq {
					minDistSq = d2
				}
			}
			d := math.Sqrt(minDistSq)
			c.pinWeight[i] = d
			if d > maxDist {
				maxDist = d
			}
		}
		if maxDist > 0 {
			invMax := 1.0 / maxDist
			for i := range vertCount {
				c.pinWeight[i] *= invMax
			}
		}
	}

	c.constraints = c.constraints[:0]
	// Horizontal structural constraints: full stretch resistance, soft fold compression down to 0.25x.
	for r := 0; r <= rows; r++ {
		rowOff := r * stride
		for col := 0; col < cols; col++ {
			a := rowOff + col
			b := a + 1
			appendClothConstraint(c, a, b, clothStructuralStiffnessScale, clothHorizMinRatio, clothHorizCompressScale)
		}
	}
	// Vertical structural constraints: full stretch resistance, soft compression down to 0.65x.
	for r := 0; r < rows; r++ {
		rowOff := r * stride
		nextOff := (r + 1) * stride
		for col := 0; col <= cols; col++ {
			a := rowOff + col
			b := nextOff + col
			appendClothConstraint(c, a, b, clothStructuralStiffnessScale, clothVertMinRatio, clothVertCompressScale)
		}
	}
	// Shear diagonal constraints: strictly tension-only (prevents diamond stretching without cantilever lock).
	for r := 0; r < rows; r++ {
		rowOff := r * stride
		nextOff := (r + 1) * stride
		for col := 0; col < cols; col++ {
			tl := rowOff + col
			tr := tl + 1
			bl := nextOff + col
			br := bl + 1
			appendClothConstraint(c, tl, br, clothShearStiffnessScale, 0, 0)
			appendClothConstraint(c, tr, bl, clothShearStiffnessScale, 0, 0)
		}
	}
	// Horizontal 2-cell bending constraints: strictly tension-only.
	for r := 0; r <= rows; r++ {
		rowOff := r * stride
		for col := 0; col+2 <= cols; col++ {
			a := rowOff + col
			b := a + 2
			appendClothConstraint(c, a, b, clothBendStiffnessScale, 0, 0)
		}
	}
	// Vertical 2-cell bending constraints: strictly tension-only.
	for r := 0; r+2 <= rows; r++ {
		rowOff := r * stride
		nextOff := (r + 2) * stride
		for col := 0; col <= cols; col++ {
			a := rowOff + col
			b := nextOff + col
			appendClothConstraint(c, a, b, clothBendStiffnessScale, 0, 0)
		}
	}

	c.cachedW = width
	c.cachedH = height
	c.initialized = true
}

func appendClothConstraint(c *ClothComponent, a, b int, stiffnessScale, minRatio, compressScale float64) {
	if c.pinned[a] && c.pinned[b] {
		return
	}
	restLen := math.Hypot(c.restX[b]-c.restX[a], c.restY[b]-c.restY[a])
	if restLen <= 0 {
		return
	}
	c.constraints = append(c.constraints, clothConstraint{
		a:              a,
		b:              b,
		restLen:        restLen,
		stiffnessScale: stiffnessScale,
		minRatio:       minRatio,
		compressScale:  compressScale,
	})
}

func isClothVertexPinned(c *ClothComponent, col, row, cols, rows int) bool {
	switch c.PinMode {
	case ClothPinLeftEdge:
		return col == 0
	case ClothPinRightEdge:
		return col == cols
	case ClothPinTopEdge:
		return row == 0
	case ClothPinTopCorners:
		return row == 0 && (col == 0 || col == cols)
	case ClothPinCustom:
		if c.PinFunc != nil {
			return c.PinFunc(col, row, cols, rows)
		}
		return false
	default:
		return col == 0
	}
}

func ensureFloat64Slice(s []float64, n int) []float64 {
	if cap(s) < n {
		return make([]float64, n)
	}
	return s[:n]
}

func ensureBoolSlice(s []bool, n int) []bool {
	if cap(s) < n {
		return make([]bool, n)
	}
	return s[:n]
}

func ensureFloat32Slice(s []float32, n int) []float32 {
	if cap(s) < n {
		return make([]float32, n)
	}
	return s[:n]
}

func ensureClothQuadDepthSlice(s []clothQuadDepth, n int) []clothQuadDepth {
	if cap(s) < n {
		return make([]clothQuadDepth, n)
	}
	return s[:n]
}

func stepClothVerlet(
	c *ClothComponent,
	cols, rows int,
	dt float64,
	perClothForces []ForceSource,
	globalForces []ForceSource,
	gusts []clothLocalGust,
) {
	c.simTime += dt
	dtSq := dt * dt
	damping := math.Pow(Clamp(c.Damping, 0.0, 1.0), dt*60.0)

	windDir := c.WindDirection
	if !windDir.IsZero() {
		windDir = windDir.Normalize()
	}
	perpX, perpY := -windDir.Y, windDir.X
	hasAmbientWind := c.WindForce != 0 && !windDir.IsZero()
	windStrengthFactor := 0.0
	if hasAmbientWind && c.Gravity.Y > 0 && windDir.X != 0 {
		windStrengthFactor = Clamp(math.Abs(c.WindForce*windDir.X)/350.0, 0.0, 1.0)
	}

	invCols := 1.0 / float64(cols)
	invRows := 1.0 / float64(rows)
	stride := cols + 1
	t1 := c.simTime * c.WindSpeed * 4.0
	t2 := c.simTime * c.WindSpeed * 7.3
	phase := c.WindPhase

	cellW := float64(c.cachedW) * invCols
	cellH := float64(c.cachedH) * invRows
	maxStep := math.Max(math.Max(cellW, cellH)*0.65, 6.0)
	maxStepSq := maxStep * maxStep

	posX, posY := c.posX, c.posY
	prevX, prevY := c.prevX, c.prevY
	restX, restY := c.restX, c.restY
	pinned := c.pinned
	pinWeight := c.pinWeight

	for r := 0; r <= rows; r++ {
		v := float64(r) * invRows
		rowOff := r * stride
		for col := 0; col <= cols; col++ {
			idx := rowOff + col
			if pinned[idx] {
				rx, ry := restX[idx], restY[idx]
				posX[idx] = rx
				posY[idx] = ry
				prevX[idx] = rx
				prevY[idx] = ry
				continue
			}

			px, py := posX[idx], posY[idx]
			accX, accY := c.Gravity.X, c.Gravity.Y
			if hasAmbientWind {
				u := float64(col) * invCols
				w := pinWeight[idx]
				wave := math.Sin(t1-u*6.5+phase) + 0.5*math.Sin(t2-u*11.0+v*3.5)
				windMag := c.WindForce * clothWindForceScale * (0.85 + 0.15*wave)
				turbMag := c.WindTurbulence * wave * w
				accX += windDir.X*windMag + perpX*turbMag
				accY += windDir.Y*windMag + perpY*turbMag

				// Aerodynamic lift raises drooping horizontal flags back toward rest height.
				if windStrengthFactor > 0 && py > restY[idx] && c.cachedH > 0 {
					droop := Clamp((py-restY[idx])/float64(c.cachedH)*3.0, 0.0, 1.0)
					accY -= c.Gravity.Y * 0.85 * windStrengthFactor * droop
				}
			}

			if len(perClothForces) > 0 {
				ax, ay := accumulateRadialForces(px, py, perClothForces)
				accX += ax
				accY += ay
			}
			if len(globalForces) > 0 {
				ax, ay := accumulateRadialForces(px, py, globalForces)
				accX += ax
				accY += ay
			}
			for g := range gusts {
				lg := &gusts[g]
				vx := px - lg.pos.X
				vy := py - lg.pos.Y
				distAlong := math.Abs(vx*lg.dir.X + vy*lg.dir.Y)
				perpDist := math.Abs(vx*lg.dir.Y - vy*lg.dir.X)
				if perpDist >= lg.halfWidth || distAlong >= lg.halfLength {
					continue
				}
				wSmooth := 1.0 - perpDist/lg.halfWidth
				lSmooth := 1.0 - distAlong/lg.halfLength
				falloff := (wSmooth * wSmooth) * (lSmooth * lSmooth)
				scale := lg.strength * lg.strengthMultiplier * falloff * clothForceScale
				accX += lg.dir.X * scale
				accY += lg.dir.Y * scale
			}

			vx := (px - prevX[idx]) * damping
			vy := (py - prevY[idx]) * damping
			vSq := vx*vx + vy*vy
			if vSq > maxStepSq {
				scale := maxStep / math.Sqrt(vSq)
				vx *= scale
				vy *= scale
			}

			dx := vx + accX*dtSq
			dy := vy + accY*dtSq
			dSq := dx*dx + dy*dy
			if dSq > maxStepSq {
				scale := maxStep / math.Sqrt(dSq)
				dx *= scale
				dy *= scale
			}

			prevX[idx] = px
			prevY[idx] = py
			posX[idx] = px + dx
			posY[idx] = py + dy
		}
	}
}

func accumulateRadialForces(px, py float64, sources []ForceSource) (ax, ay float64) {
	for i := range sources {
		fs := sources[i]
		if fs.Radius <= 0 || fs.Strength == 0 {
			continue
		}
		dx := px - fs.Position.X
		dy := py - fs.Position.Y
		distSq := dx*dx + dy*dy
		radiusSq := fs.Radius * fs.Radius
		if distSq >= radiusSq {
			continue
		}
		dist := math.Sqrt(distSq)
		smoothFalloff := 1.0 - dist/fs.Radius
		falloff := smoothFalloff * smoothFalloff * smoothFalloff
		dirX, dirY := 1.0, 0.0
		if dist > 1e-4 {
			invDist := 1.0 / dist
			dirX = dx * invDist
			dirY = dy * invDist
		}
		scale := fs.Strength * falloff * clothForceScale
		ax += dirX * scale
		ay += dirY * scale
	}
	return ax, ay
}

func solveClothConstraints(c *ClothComponent, cols, rows int, gusts []clothLocalGust) {
	iterations := c.ConstraintIterations
	if iterations <= 0 {
		iterations = defaultClothConstraintIterations
	}
	stiffness := c.Stiffness
	if stiffness <= 0 {
		stiffness = defaultClothStiffness
	} else {
		stiffness = Clamp(stiffness, 0.1, 1.0)
	}

	posX, posY := c.posX, c.posY
	prevY := c.prevY
	pinned := c.pinned
	constraints := c.constraints

	hangingTop := (c.PinMode == ClothPinTopEdge || c.PinMode == ClothPinTopCorners) && c.Gravity.Y > 0 && len(c.restY) > 0
	stride := cols + 1
	cellW, cellH := 0.0, 0.0
	if cols > 0 {
		cellW = float64(c.cachedW) / float64(cols)
	}
	if rows > 0 {
		cellH = float64(c.cachedH) / float64(rows)
	}
	var minY, minRowSep, targetRowSep float64
	if hangingTop {
		minY = c.restY[0] + 1.0
		minRowSep = cellH * 0.15
		targetRowSep = cellH * 0.25
	}

	var poleClearLeft, poleClearRight bool
	var poleSoftLimit, poleSoftTarget float64
	if cellW > 0 {
		if c.PinMode == ClothPinLeftEdge && len(c.restX) > 0 {
			allowWestSwing := c.WindForce > 0 && c.WindDirection.X < 0
			if !allowWestSwing {
				for g := range gusts {
					if gusts[g].dir.X < 0 && gusts[g].strength*gusts[g].strengthMultiplier > 0 {
						allowWestSwing = true
						break
					}
				}
			}
			if !allowWestSwing {
				poleClearLeft = true
				poleX := c.restX[0]
				poleSoftLimit = poleX - cellW*0.35
				poleSoftTarget = poleX - cellW*0.15
			}
		} else if c.PinMode == ClothPinRightEdge && len(c.restX) > cols {
			allowEastSwing := c.WindForce > 0 && c.WindDirection.X > 0
			if !allowEastSwing {
				for g := range gusts {
					if gusts[g].dir.X > 0 && gusts[g].strength*gusts[g].strengthMultiplier > 0 {
						allowEastSwing = true
						break
					}
				}
			}
			if !allowEastSwing {
				poleClearRight = true
				poleX := c.restX[cols]
				poleSoftLimit = poleX + cellW*0.35
				poleSoftTarget = poleX + cellW*0.15
			}
		}
	}

	for range iterations {
		for i := range constraints {
			con := constraints[i]
			a, b := con.a, con.b
			pinA := pinned[a]
			pinB := pinned[b]
			if pinA && pinB {
				continue
			}
			dx := posX[b] - posX[a]
			dy := posY[b] - posY[a]
			distSq := dx*dx + dy*dy
			if distSq < 1e-12 {
				continue
			}
			dist := math.Sqrt(distSq)

			var diff float64
			if dist > con.restLen {
				diff = ((con.restLen - dist) / dist) * stiffness * con.stiffnessScale
			} else if con.minRatio > 0 {
				minLen := con.restLen * con.minRatio
				if dist >= minLen {
					continue
				}
				diff = ((minLen - dist) / dist) * stiffness * con.compressScale
			} else {
				continue
			}

			if pinA {
				posX[b] += dx * diff
				posY[b] += dy * diff
			} else if pinB {
				posX[a] -= dx * diff
				posY[a] -= dy * diff
			} else {
				half := 0.5 * diff
				offX := dx * half
				offY := dy * half
				posX[a] -= offX
				posY[a] -= offY
				posX[b] += offX
				posY[b] += offY
			}
		}

		if hangingTop {
			for idx := range posY {
				if pinned[idx] {
					continue
				}
				if posY[idx] < minY {
					posY[idx] = minY
					prevY[idx] = minY
				}
			}
			if cellH > 0 {
				for r := 1; r <= rows; r++ {
					aboveOff := (r - 1) * stride
					belowOff := r * stride
					for col := 0; col <= cols; col++ {
						idxAbove := aboveOff + col
						idxBelow := belowOff + col
						if pinned[idxBelow] {
							continue
						}
						if posY[idxBelow] < posY[idxAbove]+minRowSep {
							targetY := posY[idxAbove] + targetRowSep
							posY[idxBelow] += (targetY - posY[idxBelow]) * 0.5
						}
					}
				}
			}
		}

		if poleClearLeft {
			for idx := range posX {
				if pinned[idx] {
					continue
				}
				if posX[idx] < poleSoftLimit {
					posX[idx] += (poleSoftTarget - posX[idx]) * 0.5
				}
			}
		} else if poleClearRight {
			for idx := range posX {
				if pinned[idx] {
					continue
				}
				if posX[idx] > poleSoftLimit {
					posX[idx] += (poleSoftTarget - posX[idx]) * 0.5
				}
			}
		}
	}

	// Final safety clamp for hanging cloths so no row pair remains inverted after the last pass.
	if hangingTop && cellH > 0 {
		for r := 1; r <= rows; r++ {
			aboveOff := (r - 1) * stride
			belowOff := r * stride
			for col := 0; col <= cols; col++ {
				idxAbove := aboveOff + col
				idxBelow := belowOff + col
				if pinned[idxBelow] {
					continue
				}
				minFloor := posY[idxAbove] + minRowSep
				if posY[idxBelow] < minFloor {
					posY[idxBelow] = minFloor
				}
			}
		}
	}
}

func syncClothMesh(c *ClothComponent, mesh *MeshComponent, cols, rows int) {
	verts := mesh.Vertices
	posX, posY := c.posX, c.posY
	for i := range posX {
		verts[i].DstX = float32(posX[i])
		verts[i].DstY = float32(posY[i])
	}

	sortClothQuadsBackToFront(c, mesh, cols, rows)

	if c.ShadeStrength <= 0 {
		for i := range verts {
			verts[i].ColorR = 1
			verts[i].ColorG = 1
			verts[i].ColorB = 1
			verts[i].ColorA = 1
		}
		return
	}

	updateClothShading(c, mesh, cols, rows)
}

func sortClothQuadsBackToFront(c *ClothComponent, mesh *MeshComponent, cols, rows int) {
	quadCount := cols * rows
	if quadCount <= 0 || len(c.quadOrder) != quadCount || len(c.backFaceAccum) != len(mesh.Vertices) {
		return
	}

	verts := mesh.Vertices
	clear(c.backFaceAccum)

	hasWave := c.WindForce != 0 && !c.WindDirection.IsZero()
	t1 := c.simTime * c.WindSpeed * 4.0
	t2 := c.simTime * c.WindSpeed * 7.3
	phase := c.WindPhase
	invCols := 1.0 / float64(cols)
	invRows := 1.0 / float64(rows)
	stride := cols + 1
	quads := c.quadOrder

	cellIdx := 0
	for row := 0; row < rows; row++ {
		v := (float64(row) + 0.5) * invRows
		rowOff := row * stride
		nextOff := (row + 1) * stride
		for col := 0; col < cols; col++ {
			tl := uint16(rowOff + col)
			tr := tl + 1
			bl := uint16(nextOff + col)
			br := bl + 1

			xTL, yTL := verts[tl].DstX, verts[tl].DstY
			xTR, yTR := verts[tr].DstX, verts[tr].DstY
			xBL, yBL := verts[bl].DstX, verts[bl].DstY
			xBR, yBR := verts[br].DstX, verts[br].DstY

			cross1 := (xTR-xTL)*(yBL-yTL) - (yTR-yTL)*(xBL-xTL)
			cross2 := (xTR-xBL)*(yBR-yBL) - (yTR-yBL)*(xBR-xBL)
			signedArea := float32(0.5) * (cross1 + cross2)

			var depth float32
			if signedArea < 0 {
				depth = -1000.0 + float32(row)*0.5 - float32(col)*0.25 + signedArea*0.01
				c.backFaceAccum[tl] += 1.0
				c.backFaceAccum[tr] += 1.0
				c.backFaceAccum[bl] += 1.0
				c.backFaceAccum[br] += 1.0
			} else {
				var waveZ float32
				if hasWave {
					u := (float64(col) + 0.5) * invCols
					waveZ = float32(math.Sin(t1-u*6.5+phase) + 0.5*math.Sin(t2-u*11.0+v*3.5))
				}
				switch c.PinMode {
				case ClothPinLeftEdge:
					depth = 1000.0 + float32(col)*2.0 + float32(row)*0.5 + waveZ*4.0
				case ClothPinRightEdge:
					depth = 1000.0 + float32(cols-col)*2.0 + float32(row)*0.5 + waveZ*4.0
				default:
					depth = 1000.0 + float32(row)*2.0 + waveZ*4.0
				}
			}

			quads[cellIdx] = clothQuadDepth{
				tl:    tl,
				tr:    tr,
				bl:    bl,
				br:    br,
				depth: depth,
			}
			cellIdx++
		}
	}

	// In-place insertion sort: O(N) on frame-coherent cloth quads, zero allocations.
	for i := 1; i < len(quads); i++ {
		key := quads[i]
		j := i - 1
		for j >= 0 && quads[j].depth > key.depth {
			quads[j+1] = quads[j]
			j--
		}
		quads[j+1] = key
	}

	neededIndices := quadCount * 6
	if cap(mesh.Indices) < neededIndices {
		mesh.Indices = make([]uint16, neededIndices)
	} else {
		mesh.Indices = mesh.Indices[:neededIndices]
	}
	indices := mesh.Indices
	out := 0
	for i := range quads {
		q := quads[i]
		indices[out] = q.tl
		indices[out+1] = q.tr
		indices[out+2] = q.bl
		indices[out+3] = q.tr
		indices[out+4] = q.br
		indices[out+5] = q.bl
		out += 6
	}
}

func updateClothShading(c *ClothComponent, mesh *MeshComponent, cols, rows int) {
	verts := mesh.Vertices
	posX, posY := c.posX, c.posY
	restX, restY := c.restX, c.restY
	shade := c.ShadeStrength
	stride := cols + 1

	for r := 0; r <= rows; r++ {
		rT := max(0, r-1)
		rB := min(rows, r+1)
		rowOff := r * stride
		topOff := rT * stride
		botOff := rB * stride

		for col := 0; col <= cols; col++ {
			idx := rowOff + col
			cL := max(0, col-1)
			cR := min(cols, col+1)
			idxL := rowOff + cL
			idxR := rowOff + cR
			idxT := topOff + col
			idxB := botOff + col

			var horizComp, slopeY, ridgeHoriz float64
			restDX := restX[idxR] - restX[idxL]
			if restDX > 0 {
				curDX := posX[idxR] - posX[idxL]
				curDY := posY[idxR] - posY[idxL]
				currDist := math.Hypot(curDX, curDY)
				horizComp = (restDX - currDist) / restDX
				slopeY = curDY / restDX
				if col > 0 && col < cols {
					ridgeHoriz = ((posY[idxL]+posY[idxR])*0.5 - posY[idx]) / restDX
				}
			}

			var vertComp, slopeX, ridgeVert float64
			restDY := restY[idxB] - restY[idxT]
			if restDY > 0 {
				curVX := posX[idxB] - posX[idxT]
				curVY := posY[idxB] - posY[idxT]
				currDist := math.Hypot(curVX, curVY)
				vertComp = (restDY - currDist) / restDY
				slopeX = curVX / restDY
				if r > 0 && r < rows {
					ridgeVert = (posX[idx] - (posX[idxT]+posX[idxB])*0.5) / restDY
				}
			}

			compression := horizComp + vertComp
			waveSlope := (slopeY-slopeX)*0.75 + (ridgeHoriz+ridgeVert)*2.2
			foldSignal := Clamp(-compression*1.85+waveSlope, -1.0, 1.0)
			var brightness float64
			if foldSignal < 0 {
				brightness = 1.0 + foldSignal*shade
			} else {
				brightness = 1.0 + foldSignal*(shade*0.45)
			}
			if len(c.backFaceAccum) == len(verts) && c.backFaceAccum[idx] > 0 {
				brightness -= 0.22 * shade
			}
			b32 := float32(Clamp(brightness, clothMinShade, clothMaxShade))
			verts[idx].ColorR = b32
			verts[idx].ColorG = b32
			verts[idx].ColorB = b32
			verts[idx].ColorA = 1
		}
	}
}
