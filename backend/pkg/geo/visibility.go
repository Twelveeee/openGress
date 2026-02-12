package geo

import "github.com/Twelveeee/openGress/backend/pkg/state"

type point struct {
	x float64
	y float64
}

// XY 表示平面坐标点。
type XY struct {
	X float64
	Y float64
}

// PointInBounds 判断点是否在 bounds 内。
func PointInBounds(pos state.Position, bounds state.Bounds) bool {
	if pos.Latitude < bounds.MinLat || pos.Latitude > bounds.MaxLat {
		return false
	}
	if pos.Longitude < bounds.MinLon || pos.Longitude > bounds.MaxLon {
		return false
	}
	return true
}

// LinkVisibleInBounds 判断连线是否与 bounds 相交。
func LinkVisibleInBounds(from, to state.Position, bounds state.Bounds) bool {
	if PointInBounds(from, bounds) || PointInBounds(to, bounds) {
		return true
	}
	min := point{x: bounds.MinLon, y: bounds.MinLat}
	max := point{x: bounds.MaxLon, y: bounds.MaxLat}
	topLeft := point{x: min.x, y: max.y}
	topRight := point{x: max.x, y: max.y}
	bottomLeft := point{x: min.x, y: min.y}
	bottomRight := point{x: max.x, y: min.y}

	p1 := point{x: from.Longitude, y: from.Latitude}
	p2 := point{x: to.Longitude, y: to.Latitude}

	return segmentsIntersect(p1, p2, topLeft, topRight) ||
		segmentsIntersect(p1, p2, topRight, bottomRight) ||
		segmentsIntersect(p1, p2, bottomRight, bottomLeft) ||
		segmentsIntersect(p1, p2, bottomLeft, topLeft)
}

// SegmentsIntersect 判断两条线段是否相交（含共线重叠）。
func SegmentsIntersect(p1, p2, q1, q2 XY) bool {
	return segmentsIntersect(
		point{x: p1.X, y: p1.Y},
		point{x: p2.X, y: p2.Y},
		point{x: q1.X, y: q1.Y},
		point{x: q2.X, y: q2.Y},
	)
}

func segmentsIntersect(p1, p2, q1, q2 point) bool {
	o1 := orientation(p1, p2, q1)
	o2 := orientation(p1, p2, q2)
	o3 := orientation(q1, q2, p1)
	o4 := orientation(q1, q2, p2)

	if o1 != o2 && o3 != o4 {
		return true
	}
	if o1 == 0 && onSegment(p1, q1, p2) {
		return true
	}
	if o2 == 0 && onSegment(p1, q2, p2) {
		return true
	}
	if o3 == 0 && onSegment(q1, p1, q2) {
		return true
	}
	if o4 == 0 && onSegment(q1, p2, q2) {
		return true
	}
	return false
}

func orientation(a, b, c point) int {
	val := (b.y-a.y)*(c.x-b.x) - (b.x-a.x)*(c.y-b.y)
	switch {
	case val == 0:
		return 0
	case val > 0:
		return 1
	default:
		return 2
	}
}

func onSegment(a, b, c point) bool {
	return b.x <= max(a.x, c.x) && b.x >= min(a.x, c.x) && b.y <= max(a.y, c.y) && b.y >= min(a.y, c.y)
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
