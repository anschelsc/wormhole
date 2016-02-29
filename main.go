package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"math"
	"os"
	"runtime"
	"strconv"
	"sync"

	"net/http"
)

const (
	RAD_MULT     = math.Pi / 180
	EARTH_RADIUS = 3956.6 //miles
)

const (
	PLACES = "places.csv"
	LOG = "wormhole.log"
)

var (
	all []*place
	logger *log.Logger
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

func find(city, state string) *place {
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

func findPath(start, end *place) (startRef, endRef *placeRef) {
	unvisited := make([]*placeRef, 0, len(all)-1)
	startRef = &placeRef{place: start, dist: 0, next: nil}
	if start == end {
		endRef = startRef
		return
	}
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

func getPath(end *placeRef) []*place {
	path := make([]*place, 0)
	for p := end; p != nil; p = p.next {
		path = append(path, p.place)
	}
	return path
}

func handleRoute(w http.ResponseWriter, r *http.Request) {
	// DON'T PUT THIS IN PRODUCTION
	w.Header().Add("Access-Control-Allow-Origin", "*")

	encoder := json.NewEncoder(w)
	query := r.URL.Query()
	start := find(query.Get("startCity"), query.Get("startState"))
	if start == nil {
		logger.Println("Bad start")
		encoder.Encode("Bad start")
		return
	}
	end := find(query.Get("endCity"), query.Get("endState"))
	if end == nil {
		logger.Println("Bad end")
		encoder.Encode("Bad end")
		return
	}
	_, startRef := findPath(end, start)
	path := getPath(startRef)
	err := encoder.Encode(path)
	if err != nil {
		logger.Println(err)
		return
	}
}

func main() {
	logFile, err := os.OpenFile(LOG, os.O_APPEND | os.O_CREATE | os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()
	logger = log.New(logFile, "", log.Ldate | log.Ltime | log.Lshortfile | log.LUTC)

	file, err := os.Open(PLACES)
	if err != nil {
		logger.Println(os.Stderr, err)
		return
	}
	defer file.Close()
	fbuf := bufio.NewReader(file)

	all = make([]*place, 0)
	for {
		p, err := readPlace(fbuf)
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Println(os.Stderr, err)
			return
		}
		all = append(all, p)
	}

	http.HandleFunc("/", handleRoute)
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		logger.Println(err)
	}
}
