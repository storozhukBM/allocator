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

type CirclePtr struct {
	c     *Circle
	Color uint64
}

type CircleCirclePtr struct {
	cp    CirclePtr
	Color uint64
}

type FixedCircleCirclePtrVector struct {
	circles [3]CircleCirclePtr
}
