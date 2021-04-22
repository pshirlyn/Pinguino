package pinguino

import (
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"pinguino/src/labrpc"
	"sync"
	"time"
)

const heartbeatTimeInterval = time.Duration(10 * time.Millisecond)

type Coordinator struct {
	mu sync.Mutex

	nRegions          int
	workers           []*labrpc.ClientEnd
	lastHeartbeats    []time.Time
	playerToRegionMap map[string]int
	regionToWorkerMap map[int]int

	killed bool
}

// Used to pick a region to assign to players.
// For now, pick the region randomly. In the future, consider load balancing.
func (c *Coordinator) pickRegion() int {
	return rand.Intn(c.nRegions)
}

func (c *Coordinator) AssignPlayerToRegion(args *AssignPlayerToRegionArgs, reply *AssignPlayerToRegionReply) {
	c.mu.Lock()
	defer c.mu.Unlock()

	username := args.Username
	region := c.pickRegion()
	c.playerToRegionMap[username] = region

	reply.Success = true
	reply.Region = region
	reply.Worker = c.regionToWorkerMap[region]
}

func (c *Coordinator) sendHeartbeatToWorker(workerIndex int, args *HeartbeatArgs, reply *HeartbeatReply) {
	ok := c.workers[workerIndex].Call("Worker.Heartbeat", &args, &reply)

	if !ok {
		log.Println("couldn't reach worker")
		// TODO: handle worker disconnect
		return
	}

	// Successfully sent heartbeat to worker, so update lastHearbeat
	c.lastHeartbeats[workerIndex] = time.Now()
}

func (c *Coordinator) maybeSendHeartbeats() {
	for i := 0; i < len(c.workers); i++ {
		args := HeartbeatArgs{}
		reply := HeartbeatReply{}
		// Check time since last hearbeat
		if time.Since(c.lastHeartbeats[i]) > heartbeatTimeInterval {
			go c.sendHeartbeatToWorker(i, &args, &reply)
		}
	}
}

// to implement:
// func (c *Coordinator) MovePlayer()

//
// start a thread that listens for RPCs from worker.go
//
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

func (c *Coordinator) Kill() {
}

func (c *Coordinator) run() {
	// main loop
	for !c.killed {
		c.maybeSendHeartbeats()
		time.Sleep(heartbeatTimeInterval)
	}
}

func MakeCoordinator(workers []*labrpc.ClientEnd, regions int) *Coordinator {
	c := &Coordinator{}

	c.mu.Lock()
	defer c.mu.Unlock()

	// TODO: add coordinator backup server reference here
	c.nRegions = regions
	c.killed = false

	c.workers = workers
	c.lastHeartbeats = make([]time.Time, len(c.workers))
	// Initialize all last heartbeats to current time.
	for i := 0; i < len(c.workers); i++ {
		c.lastHeartbeats[i] = time.Now()
	}

	c.playerToRegionMap = make(map[string]int)
	c.regionToWorkerMap = make(map[int]int)

	// Assign main worker to each region.
	// TODO: handle cases where the number of regions != number of workers
	// TODO: perhaps randomize the assignment
	for i := 0; i < c.nRegions; i++ {
		c.regionToWorkerMap[i] = i

	}

	go c.run()
	return c
}
