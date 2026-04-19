package docker

// SystemMemory holds host memory statistics.
type SystemMemory struct {
	Total     uint64
	Used      uint64
	Available uint64
	Percent   float64
}

// SystemLoad holds host load averages.
type SystemLoad struct {
	Load1  float64
	Load5  float64
	Load15 float64
}
