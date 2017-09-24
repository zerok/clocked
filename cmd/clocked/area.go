package main

type Area struct {
	X      int
	Y      int
	Width  int
	Height int
}

func (a *Area) XMin() int {
	return a.X
}

func (a *Area) YMin() int {
	return a.Y
}

func (a *Area) XMax() int {
	return a.X + a.Width - 1
}

func (a *Area) YMax() int {
	return a.Y + a.Height - 1
}
