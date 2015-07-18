package main

import (
	"fmt"

	"time"

	"math/rand"

	"encoding/json"

	"github.com/achilleasa/usrv"
	"github.com/achilleasa/usrv-tracer"
	"github.com/achilleasa/usrv-tracer/middleware"
	"github.com/achilleasa/usrv-tracer/storage"
	"github.com/achilleasa/usrv/usrvtest"
	"golang.org/x/net/context"
)

type Add4Request struct {
	A, B, C, D int
}
type Add2Request struct {
	A, B int
}
type AddResponse struct {
	Sum int
}

type Adder struct {
	add2Client *usrv.Client
	add4Client *usrv.Client
	Server     *usrv.Server
}

// add 2 numbers
func (adder *Adder) add2(ctx context.Context, rawReq interface{}) (interface{}, error) {
	req := rawReq.(Add2Request)

	// Simulate processing delay
	<-time.After(time.Millisecond * time.Duration(rand.Intn(5)))

	return &AddResponse{Sum: req.A + req.B}, nil
}

// Add 4 numbers. Makes 2 parallel calls to add2 and sums the results
func (adder *Adder) add4(ctx context.Context, rawReq interface{}) (interface{}, error) {
	req := rawReq.(Add4Request)

	// Add (a,b) and (c,d) in parallel
	req1, _ := json.Marshal(Add2Request{A: req.A, B: req.B})
	req1Chan := adder.add2Client.Request(
		ctx, // Make sure you include the original context so requests can be linked together
		&usrv.Message{Payload: req1},
	)

	req2, _ := json.Marshal(Add2Request{A: req.C, B: req.D})
	req2Chan := adder.add2Client.Request(
		ctx, // Make sure you include the original context so requests can be linked together
		&usrv.Message{Payload: req2},
	)

	// Wait for responses
	var res1, res2 AddResponse
	for {
		select {
		case srvRes := <-req1Chan:
			json.Unmarshal(srvRes.Message.Payload, &res1)
			req1Chan = nil
		case srvRes := <-req2Chan:
			json.Unmarshal(srvRes.Message.Payload, &res2)
			req2Chan = nil
		}

		if req1Chan == nil && req2Chan == nil {
			break
		}
	}

	// Run a final add2 to get the sum
	req3, _ := json.Marshal(Add2Request{A: res1.Sum, B: res2.Sum})
	req3Chan := adder.add2Client.Request(
		ctx, // Make sure you include the original context so requests can be linked together
		&usrv.Message{Payload: req3},
	)

	srvRes := <-req3Chan
	var res3 AddResponse
	json.Unmarshal(srvRes.Message.Payload, &res3)
	return &res3, nil
}

func (adder *Adder) Add4(a, b, c, d int) (int, string) {
	req, _ := json.Marshal(Add4Request{a, b, c, d})
	reqChan := adder.add4Client.Request(
		context.WithValue(context.Background(), usrv.CtxCurEndpoint, "com.test.api"),
		&usrv.Message{Payload: req},
	)

	srvRes := <-reqChan
	var res AddResponse
	json.Unmarshal(srvRes.Message.Payload, &res)

	// Return the value and the injected trace id
	return res.Sum, srvRes.Message.Headers.Get(middleware.CtxTraceId).(string)
}

func NewAdder(transp usrv.Transport, collector *tracer.Collector) *Adder {
	server, err := usrv.NewServer(transp)
	if err != nil {
		panic(err)
	}

	add2Dec := func(data []byte) (interface{}, error) {
		var payload Add2Request
		err := json.Unmarshal(data, &payload)
		return payload, err
	}

	add4Dec := func(data []byte) (interface{}, error) {
		var payload Add4Request
		err := json.Unmarshal(data, &payload)
		return payload, err
	}

	adder := &Adder{
		add2Client: usrv.NewClient(transp, "com.test.add/2"),
		add4Client: usrv.NewClient(transp, "com.test.add/4"),
		Server:     server,
	}

	// Register endpoints and add the tracer middleware
	server.Handle(
		"com.test.add/2",
		usrv.PipelineHandler{add2Dec, adder.add2, json.Marshal},
		middleware.Tracer(collector),
	)
	server.Handle(
		"com.test.add/4",
		usrv.PipelineHandler{add4Dec, adder.add4, json.Marshal},
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
