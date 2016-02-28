package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"strconv"
	"sync"
)

const (
	RAD_MULT     = math.Pi / 180
	EARTH_RADIUS = 3956.6 //miles
)

type place struct {
	City, State string
	lat, lon    float64
}

type placeRef struct {
	*place
	dist float64
	next *placeRef
}

func readPlace(rd *bufio.Reader) (*place, error) {
	p := new(place)
	var err error
	p.City, err = rd.ReadString(',')
	if err != nil {
		return p, err
	}
	p.City = p.City[1 : len(p.City)-2]
	p.State, err = rd.ReadString(',')
	if err != nil {
		return p, err
	}
	p.State = p.State[1 : len(p.State)-2]
	latS, err := rd.ReadString(',')
	if err != nil {
		return p, err
	}
	p.lat, err = strconv.ParseFloat(latS[:len(latS)-1], 64)
	if err != nil {
		return p, err
	}
	p.lat *= RAD_MULT
	longS, err := rd.ReadString('\n')
	if err != nil {
		return p, err
	}
	p.lon, err = strconv.ParseFloat(longS[:len(longS)-1], 64)
	p.lon *= RAD_MULT
	return p, err
}

func dist(p1 *place, p2 *place) float64 {
	// Copied from http://www.movable-type.co.uk/scripts/latlong.html
	dLat := p2.lat - p1.lat
	dLon := p2.lon - p1.lon
	sinHalfDLat := math.Sin(dLat / 2)
	sinHalfDLon := math.Sin(dLon / 2)
	a := sinHalfDLat*sinHalfDLat + sinHalfDLon*sinHalfDLon*math.Cos(p1.lat)*math.Cos(p2.lat)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return c * EARTH_RADIUS
}

// Will anyone care if we use equirectangular approximations for the (very short) distances that actually show up here?
func fastDist(p1 *place, p2 *place) float64 {
	if p1.City == p2.City {
		return .01
	}
	// Copied from http://www.movable-type.co.uk/scripts/latlong.html
	dx := (p2.lon - p1.lon) * math.Cos((p1.lat+p2.lat)/2)
	dy := (p2.lat - p1.lat)
	return math.Sqrt(dx*dx+dy*dy) * EARTH_RADIUS
}

func find(all []*place, city, state string) *place {
	for i, p := range all {
		if p.City == city && p.State == state {
			return all[i]
		}
	}
	return nil
}

func popClosest(ps *[]*placeRef) *placeRef {
	closest := 0
	for i := 1; i < len(*ps); i++ {
		if (*ps)[i].dist < (*ps)[closest].dist {
			closest = i
		}
	}
	ret := (*ps)[closest]
	(*ps)[closest] = (*ps)[len(*ps)-1]
	*ps = (*ps)[:len(*ps)-1]
	return ret
}

func findPath(all []*place, start, end *place) (startRef, endRef *placeRef) {
	unvisited := make([]*placeRef, 0, len(all)-1)
	startRef = &placeRef{place: start, dist: 0, next: nil}
	for i := range all {
		p := all[i]
		if p != start {
			ref := &placeRef{
				place: p,
				dist:  fastDist(start, p),
				next:  startRef,
			}
			unvisited = append(unvisited, ref)
			if p == end {
				endRef = ref
			}
		}
	}
	nWorkers := runtime.GOMAXPROCS(0)
	var wg sync.WaitGroup
	for closest := popClosest(&unvisited); closest.place != end; closest = popClosest(&unvisited) {
		for i := 0; i < nWorkers; i++ {
			wg.Add(1)
			chunksize := len(unvisited)/nWorkers + 1
			go func(i int) {
				top := (i + 1) * chunksize
				if top > len(unvisited) {
					top = len(unvisited)
				}
				for _, p := range unvisited[i*chunksize : top] {
					d := closest.dist + fastDist(closest.place, p.place)
					if d < p.dist {
						p.dist = d
						p.next = closest
					}
				}
				wg.Done()
			}(i)
		}
		wg.Wait()
	}
	return
}

func printPath(p1, p2 *placeRef) {
	path := getPath(p2)
	fmt.Printf("Start in %s, %s\n", p2.City, p2.State)
	for i := 1; i < len(path); i++ {
		from, to := path[i-1], path[i]
		if from.City == to.City {
			fmt.Printf("Travel by wormhole to %s, %s\n", to.City, to.State)
		} else {
			fmt.Printf("Fly by crow to %s, %s (%.0f miles)\n", to.City, to.State, dist(from, to))
		}
	}
	fmt.Printf("Total distance was %.0f miles, compared to %.0f miles directly by crow.\n", p2.dist, dist(p1.place, p2.place))
}

func getPath(end *placeRef) []*place {
	path := make([]*place, 0)
	for p := end; p != nil; p = p.next {
		path = append(path, p.place)
	}
	return path
}

func main() {
	filename := os.Args[1]
	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer file.Close()
	fbuf := bufio.NewReader(file)

	ps := make([]*place, 0)
	for {
		p, err := readPlace(fbuf)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		ps = append(ps, p)
	}

	p2 := find(ps, os.Args[2], os.Args[3])
	p1 := find(ps, os.Args[4], os.Args[5])
	p1Ref, p2Ref := findPath(ps, p1, p2)
	printPath(p1Ref, p2Ref)
}
