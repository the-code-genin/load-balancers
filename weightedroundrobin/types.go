package weightedroundrobin

type component struct {
	id         string
	weight     int
	lowerBound int
	upperBound int
}
