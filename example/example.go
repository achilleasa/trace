package main

import (
	"fmt"
	"strconv"
	"strings"

	"time"

	"math/rand"

	"github.com/achilleasa/usrv"
	"github.com/achilleasa/usrv-tracer"
	"github.com/achilleasa/usrv-tracer/middleware"
	"github.com/achilleasa/usrv-tracer/storage"
	"github.com/achilleasa/usrv/usrvtest"
	"golang.org/x/net/context"
)

type Adder struct {
	add2Client *usrv.Client
	add4Client *usrv.Client
	Server     *usrv.Server
}

// add 2 numbers
func (adder *Adder) add2(ctx context.Context, rw usrv.ResponseWriter, msg *usrv.Message) {
	tokens := strings.Split(string(msg.Payload), " ")
	a, _ := strconv.Atoi(tokens[0])
	b, _ := strconv.Atoi(tokens[1])

	// Simulate processing delay
	<-time.After(time.Millisecond * time.Duration(rand.Intn(5)))

	// Send response
	rw.Write([]byte(fmt.Sprintf("%d", a+b)))
}

// Add 4 numbers. Makes 2 parallel calls to add2 and sums the results
func (adder *Adder) add4(ctx context.Context, rw usrv.ResponseWriter, msg *usrv.Message) {
	tokens := strings.Split(string(msg.Payload), " ")
	a, _ := strconv.Atoi(tokens[0])
	b, _ := strconv.Atoi(tokens[1])
	c, _ := strconv.Atoi(tokens[2])
	d, _ := strconv.Atoi(tokens[3])

	req1Chan := adder.add2Client.Request(
		ctx, // Make sure you include the original context so requests can be linked together
		&usrv.Message{Payload: []byte(fmt.Sprintf("%d %d", a, b))},
	)

	req2Chan := adder.add2Client.Request(
		ctx, // Make sure you include the original context so requests can be linked together
		&usrv.Message{Payload: []byte(fmt.Sprintf("%d %d", c, d))},
	)

	// Wait for responses
	var res1, res2 usrv.ServerResponse
	for {
		select {
		case res1 = <-req1Chan:
			req1Chan = nil
		case res2 = <-req2Chan:
			req2Chan = nil
		}

		if req1Chan == nil && req2Chan == nil {
			break
		}
	}

	// Run a final add2 to get the sum
	a, _ = strconv.Atoi(string(res1.Message.Payload))
	b, _ = strconv.Atoi(string(res2.Message.Payload))
	req3Chan := adder.add2Client.Request(
		ctx, // Make sure you include the original context so requests can be linked together
		&usrv.Message{Payload: []byte(fmt.Sprintf("%d %d", a, b))},
	)

	res3 := <-req3Chan
	rw.Write(res3.Message.Payload)
}

func (adder *Adder) Add4(a, b, c, d int) (int, string) {
	reqChan := adder.add4Client.Request(
		context.WithValue(context.Background(), usrv.CtxCurEndpoint, "com.test.api"),
		&usrv.Message{Payload: []byte(fmt.Sprintf("%d %d %d %d", a, b, c, d))},
	)

	res := <-reqChan
	a, _ = strconv.Atoi(string(res.Message.Payload))

	// Return the value and the injected trace id
	return a, res.Message.Headers.Get(middleware.CtxTraceId).(string)
}

func NewAdder(transp usrv.Transport, collector *tracer.Collector) *Adder {
	server, err := usrv.NewServer(transp)
	if err != nil {
		panic(err)
	}

	adder := &Adder{
		add2Client: usrv.NewClient(transp, "com.test.add/2"),
		add4Client: usrv.NewClient(transp, "com.test.add/4"),
		Server:     server,
	}

	// Register endpoints and add the tracer middleware
	server.Handle(
		"com.test.add/2",
		usrv.HandlerFunc(adder.add2),
		middleware.Tracer(collector),
	)
	server.Handle(
		"com.test.add/4",
		usrv.HandlerFunc(adder.add4),
		middleware.Tracer(collector),
	)

	return adder
}

func main() {
	// Setup collector
	collector, err := tracer.NewCollector(storage.Memory, 100, 0)
	if err != nil {
		panic(err)
	}

	// Use in-memory transport for this demo
	transp := usrvtest.NewTransport()
	defer transp.Close()

	// Start RPC server
	adder := NewAdder(transp, collector)
	go func() {
		err := adder.Server.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()
	defer adder.Server.Close()
	<-time.After(100 * time.Millisecond)

	// Make a call to the service
	sum, traceId := adder.Add4(1, 3, 5, 7)
	fmt.Printf("[%s] Sum: 1 + 3 + 5 + 7 = %d\n", traceId, sum)

	// Get trace log from storage
	traceLog, err := storage.Memory.GetTrace(traceId)
	if err != nil {
		panic(err)
	}
	fmt.Printf("\nTrace log:\n")
	for _, rec := range traceLog {
		fmt.Printf("   %v\n", rec)
	}

	// Get service dependencies
	deps, err := storage.Memory.GetDependencies()
	if err != nil {
		panic(err)
	}
	fmt.Printf("\nService dependencies:\n")
	for _, srv := range deps {
		fmt.Printf("   [%s] depends on: %s\n", srv.Service, srv.Dependencies)
	}
}
