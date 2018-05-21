package chart

// Box represents the main 4 dimensions of a box.
type Bubble struct {
	Radius    int
	MidPointX   int
	MidPointY int
	IsSet  bool
}

// Clone returns a new copy of the box.
func (b Bubble) Clone() Bubble {
	return Bubble{
		IsSet:  b.IsSet,
		MidPointX:    b.MidPointX,
		MidPointY:   b.MidPointY,
		Radius:  b.Radius,
	}
}

