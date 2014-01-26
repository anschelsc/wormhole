package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
)

const LINES = 24368

const (
	RAD_MULT     = math.Pi / 180
	EARTH_RADIUS = 3956.6 //miles
)

type place struct {
	city, state string
	lat, lon    float64
	dist        float64
	from        *place
}

func (p *place) readLine(rd *bufio.Reader) error {
	var err error
	p.city, err = rd.ReadString(',')
	if err != nil {
		return err
	}
	p.city = p.city[1 : len(p.city)-2]
	p.state, err = rd.ReadString(',')
	if err != nil {
		return err
	}
	p.state = p.state[1 : len(p.state)-2]
	latS, err := rd.ReadString(',')
	if err != nil {
		return err
	}
	p.lat, err = strconv.ParseFloat(latS[:len(latS)-1], 64)
	if err != nil {
		return err
	}
	p.lat *= RAD_MULT
	longS, err := rd.ReadString('\n')
	if err != nil {
		return err
	}
	p.lon, err = strconv.ParseFloat(longS[:len(longS)-1], 64)
	p.lon *= RAD_MULT
	return err
}

func dist(p1 *place, p2 *place) float64 {
	if p1.city == p2.city {
		return .01
	}
	// Copied from http://www.movable-type.co.uk/scripts/latlong.html
	dLat := p2.lat - p1.lat
	dLon := p2.lon - p1.lon
	sinHalfDLat := math.Sin(dLat / 2)
	sinHalfDLon := math.Sin(dLon / 2)
	a := sinHalfDLat*sinHalfDLat + sinHalfDLon*sinHalfDLon*math.Cos(p1.lat)*math.Cos(p2.lat)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return c * EARTH_RADIUS
}

func find(all []place, city, state string) *place {
	for i, p := range all {
		if p.city == city && p.state == state {
			return &all[i]
		}
	}
	return nil
}

func popClosest(ps *[]*place) *place {
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

func findPath(all []place, start, end *place) {
	unvisited := make([]*place, 0, LINES-1)
	for i := range all {
		if &all[i] != start {
			all[i].dist = dist(start, &all[i])
			all[i].from = start
			unvisited = append(unvisited, &all[i])
		}
	}
	for closest := popClosest(&unvisited); closest != end; closest = popClosest(&unvisited) {
		for _, p := range unvisited {
			d := closest.dist + dist(closest, p)
			if d < p.dist {
				p.dist = d
				p.from = closest
			}
		}
	}
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
	ps := make([]place, LINES)
	for i := range ps {
		ps[i].readLine(fbuf)
	}
	p2 := find(ps, os.Args[2], os.Args[3])
	p1 := find(ps, os.Args[4], os.Args[5])
	findPath(ps, p1, p2)
	for here := p2; here != nil; here = here.from {
		fmt.Printf("%s, %s\n", here.city, here.state)
	}
	fmt.Printf("Total distance was %.0f miles, compared to %.0f miles directly.\n", p2.dist, dist(p1, p2))
}
