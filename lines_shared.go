package katsu2d

func catmullRomPoint(p0, p1, p2, p3 Vector, t float64) Vector {
	t2 := t * t
	t3 := t2 * t

	c1 := p1.ScaleF(2)
	c2 := p2.Sub(p0).ScaleF(t)
	c3 := p0.ScaleF(2).Sub(p1.ScaleF(5)).Add(p2.ScaleF(4)).Sub(p3).ScaleF(t2)
	c4 := p0.ScaleF(-1).Add(p1.ScaleF(3)).Sub(p2.ScaleF(3)).Add(p3).ScaleF(t3)

	return c1.Add(c2).Add(c3).Add(c4).ScaleF(0.5)
}
