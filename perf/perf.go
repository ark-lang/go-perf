package perf

import (
	"fmt"
	"runtime"
	"sort"
	"time"
)

type Callsite struct {
	pc   uintptr
	file string
	line int
}

type Handle int

type Context struct {
	handleCount int
	handles     map[Callsite]Handle

	hits      map[Handle]uint64
	timeSpent map[Handle]int64
	entryTime map[Handle][]int64
}

var context *Context

func Init() {
	context = &Context{
		handleCount: 0,
		handles:     make(map[Callsite]Handle),
		hits:        make(map[Handle]uint64),
		timeSpent:   make(map[Handle]int64),
		entryTime:   make(map[Handle][]int64),
	}
}

type TimedCallsite struct {
	Callsite
	hits      uint64
	timeSpent int64
}

type TimedCallsites []TimedCallsite

func (t TimedCallsites) Len() int           { return len(t) }
func (t TimedCallsites) Less(i, j int) bool { return t[i].timeSpent > t[j].timeSpent }
func (t TimedCallsites) Swap(i, j int)      { t[j], t[i] = t[i], t[j] }

func Finalize() {
	callsites := make(TimedCallsites, len(context.handles))
	for callsite, handle := range context.handles {
		hits, timeSpent := context.hits[handle], context.timeSpent[handle]
		callsites[handle] = TimedCallsite{Callsite: callsite, hits: hits, timeSpent: timeSpent}
	}
	sort.Sort(callsites)

	for _, callsite := range callsites {
		fn := runtime.FuncForPC(callsite.pc)

		hits := callsite.hits
		timeSpent := callsite.timeSpent

		micros := float64(timeSpent) / float64(time.Microsecond)
		ratio := float64(micros) / float64(hits)

		fmt.Printf("%s(): %d hits, %.2f micros, %.2f micros/hit\n",
			fn.Name(), hits, micros, ratio)
	}
}

func (c *Context) HandleFromCallsite(pc uintptr, file string, line int) Handle {
	callsite := Callsite{pc, file, line}
	handle, ok := c.handles[callsite]
	if !ok {
		handle = Handle(c.handleCount)
		c.handleCount++
		c.handles[callsite] = handle
	}

	return handle
}

func Enter() Handle {
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		panic("go-perf: Unable to identify caller")
	}

	handle := context.HandleFromCallsite(pc, file, line)
	context.hits[handle]++
	context.entryTime[handle] = append(context.entryTime[handle], time.Now().UnixNano())
	return handle
}

func Exit(handle Handle) {
	now := time.Now().UnixNano()

	entryTimes := context.entryTime[handle]
	idx := len(entryTimes) - 1
	context.entryTime[handle] = entryTimes[:idx]

	entryTime := entryTimes[idx]
	context.timeSpent[handle] += now - entryTime
}
