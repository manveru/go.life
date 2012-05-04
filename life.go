// Conway's Game of Life.

package main

import (
	"flag"
	"github.com/banthar/Go-SDL/sdl"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// all the flags
var Rule *string = flag.String("rule", "B3/S23", "Rule")
var aliveColor *string = flag.String("alive-color", "ffffff", "Color of alive cells")
var deadColor *string = flag.String("dead-color", "000000", "Color of dead cells")
var worldWidth *int = flag.Int("width", 160, "Amount of horizontal cells")
var worldHeight *int = flag.Int("height", 120, "Amount of vertical cells")
var delay *int = flag.Int("delay", 25, "milliseconds between iterations")
var scale *int = flag.Int("scale", 4, "scale world by factor N")

// setting up variables used across most functions
var (
	AliveColor, DeadColor, Delay uint32
	Cells                        *[][]bool
	Paused, Running              bool
	Rects                        *[][]*sdl.Rect
	Screen                       *sdl.Surface
	Height, Scale, Width         int
)

var rule = map[bool]map[int]bool{
	true: map[int]bool{}, false: map[int]bool{},
}

func init() {
	flag.Parse()

	Delay = uint32(*delay)
	Scale = *scale

	color, err := strconv.ParseUint(*aliveColor, 16, 32)
	if err == nil {
		AliveColor = uint32(color)
	} else {
		panic(err)
	}

	color, err = strconv.ParseUint(*deadColor, 16, 32)
	if err == nil {
		DeadColor = uint32(color)
	} else {
		panic(err)
	}

	Width = *worldWidth
	Height = *worldHeight

	for _, part := range strings.SplitN(*Rule, "/", 2) {
		name, rest := string(part[0]), part[1:]
		for _, s := range strings.Split(rest, "") {
			n, err := strconv.Atoi(s)
			if err != nil {
				panic(err)
			}

			switch name {
			case "B": // Born
				rule[false][n] = true
			case "S": // Survive
				rule[true][n] = true
			}
		}
	}
}

func main() {
	if sdl.Init(sdl.INIT_EVERYTHING) != 0 {
		panic(sdl.GetError())
	}
	defer sdl.Quit()

	println("unpause with space or p")
	println("while paused you can place cells with the three mouse buttons")

	rand.Seed(time.Now().UnixNano())

	Screen = NewSurface(Width*Scale, Height*Scale)
	Setup()
	HandleEvents()
}

func Setup() {
	rectsp, cellsp := MakeRects(), MakeCells()
	rects, cells := *rectsp, *cellsp

	for x, l := range cells {
		for y, _ := range l {
			rx, ry := x*Scale, y*Scale
			wh := uint16(Scale)
			rect := &sdl.Rect{X: int16(rx), Y: int16(ry), W: wh, H: wh}
			rects[x][y] = rect

			// uncomment to seed the game field with random life
			// if rand.Float() > 0.95 { cells[x][y] = true }
		}
	}
	Rects = rectsp
	Cells = cellsp

	// AddAcorn(100, 100)
	AddGlider(Width/2, Height/2)
	// AddDiehard(100, 100)
	DrawCells()

	Running = true
	Paused = true
}

func HandleEvents() {
	ch := make(chan func(), 10)

	for Running {
		for ev := sdl.PollEvent(); ev != nil; ev = sdl.PollEvent() {
			switch e := ev.(type) {
			case *sdl.QuitEvent:
				Running = false
			case *sdl.MouseButtonEvent:
				if e.State == 1 {
					x, y := int(e.X), int(e.Y)
					switch e.Button {
					case 1:
						queue(ch, func() { ToggleCell(x/Scale, y/Scale) })
					case 2:
						queue(ch, func() { AddAcorn(x/Scale, y/Scale) })
					case 3:
						queue(ch, func() { AddGlider(x/Scale, y/Scale) })
					}
				}
			case *sdl.KeyboardEvent:
				if e.State == 1 {
					switch e.Keysym.Sym {
					case sdl.K_ESCAPE:
						Running = false
					case sdl.K_SPACE, sdl.K_p:
						TogglePause(ch)
					}
				}
			}
		}

		sdl.Delay(25)
	}
}

func queue(ch chan func(), fun func()) {
	if !Running {
		return
	}
	if Paused {
		fun()
		DrawCells()
	} else {
		ch <- fun
	}
}

func TogglePause(ch chan func()) {
	if Paused {
		Paused = false
		go RefreshCells(ch)
	} else {
		Paused = true
	}
}

func ToggleCell(x, y int) {
	(*Cells)[x][y] = !(*Cells)[x][y]
}

// Wrap around world edges.
func check(x, y int) int {
	cells := *Cells
	// force x into world boundaries
	if x >= 0 {
		if len(cells) > x {
			// ok
		} else {
			x = len(cells) - x
		}
	} else {
		x = len(cells) + x
	}

	line := cells[x]

	if y >= 0 {
		if len(line) > y {
			// ok
		} else {
			y = len(line) - y
		}
	} else {
		y = len(line) + y
	}

	if line[y] {
		return 1
	}
	return 0
}

func Count(x, y int) int {
	return check(x-1, y-1) +
		check(x, y-1) +
		check(x+1, y-1) +
		check(x-1, y) +
		check(x+1, y) +
		check(x-1, y+1) +
		check(x, y+1) +
		check(x+1, y+1)
}

func MakeRects() *[][]*sdl.Rect {
	rects := make([][]*sdl.Rect, Width)
	for i, _ := range rects {
		rects[i] = make([]*sdl.Rect, Height)
	}
	return &rects
}

func MakeCells() *[][]bool {
	cells := make([][]bool, Width)
	for i, _ := range cells {
		cells[i] = make([]bool, Height)
	}
	return &cells
}

func RefreshCells(ch chan func()) {
	var targetp *[][]bool
	var target [][]bool
	var l []bool
	var alive bool
	var x, y, count int

	for Running && !Paused {
		targetp = MakeCells()
		target = *targetp

		for x, l = range *Cells {
			for y, alive = range l {
				DrawCell(x, y, alive)

				count = Count(x, y)
				target[x][y] = rule[alive][count]
			}
		}

		select {
		case fun := <-ch:
			fun()
		default:
			break
		}

		Screen.Flip()
		sdl.Delay(Delay)
		Cells = targetp
	}
}

func DrawCells() {
	for x, l := range *Cells {
		for y, alive := range l {
			DrawCell(x, y, alive)
		}
	}

	Screen.Flip()
}

func DrawCell(x, y int, c bool) {
	rect := (*Rects)[x][y]
	if c {
		Screen.FillRect(rect, AliveColor)
	} else {
		Screen.FillRect(rect, DeadColor)
	}
}

func NewSurface(height int, width int) (surface *sdl.Surface) {
	surface = sdl.SetVideoMode(height, width, 32, 0)
	if surface == nil {
		panic(sdl.GetError())
	}
	return
}

// Add Glider:
// + 0 1 2
//
// 0 O O
//
// 1 O   O
//
// 2 O

func AddGlider(x, y int) {
	cells := *Cells
	cells[x+2][y+2] = true
	cells[x+2][y+3] = true
	cells[x+3][y+2] = true
	cells[x+4][y+2] = true
	cells[x+3][y+4] = true
}

// Add Acorn:
//  + 0 1 2 3 4 5 6
//
//  0   O
//
//  1       O
//
//  2 O O     O O O

func AddAcorn(x, y int) {
	cells := *Cells
	cells[x+1][y+0] = true
	cells[x+3][y+1] = true
	cells[x+0][y+2] = true
	cells[x+1][y+2] = true
	cells[x+4][y+2] = true
	cells[x+5][y+2] = true
	cells[x+6][y+2] = true
}

// Add Diehard:
//    y
//    |
//x - + 0 1 2 3 4 5 6 7
//    0             O
//
//    1 O O
//
//    2   O       O O O
func AddDiehard(x, y int) {
	cells := *Cells
	cells[x+0][y+1] = true
	cells[x+1][y+1] = true
	cells[x+1][y+2] = true
	cells[x+5][y+2] = true
	cells[x+6][y+0] = true
	cells[x+6][y+2] = true
	cells[x+7][y+2] = true
}
