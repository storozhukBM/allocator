package etalon

type coordinate int

type coordinates []coordinate

type Point struct {
	X int32
	Y int32
}

type PointsVector struct {
	points []Point
}

type StablePointsVector struct {
	Points [3]Point
}

type Circle struct {
	center Point
	radius int
}

type CircleColor struct {
	Circle
	Color uint64
}

type CircleWithPointer struct {
	c     *Circle
	Color uint64
}

type EmbeddedCircleWithPointer struct {
	cp    CircleWithPointer
	Color uint64
}

type FixedEmbeddedCircleWithPointerVector struct {
	circles [3]EmbeddedCircleWithPointer
}
