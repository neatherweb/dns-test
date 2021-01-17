/*
Copyright 2020
Author: Jason Neatherway @neatherweb
*/
package main

import (
		"encoding/csv"
		"encoding/json"
		"flag"
		"fmt"
		"io"
		"os"
		"strings"
		"sync"
		"time"
		"github.com/miekg/dns"
		log "github.com/sirupsen/logrus"
		"github.com/dimchansky/utfbom"
)
var (
	version = "0.3"
	inputfile = flag.String("input", "", "CSV input file for queries, rows formatted as <type>,<name_to_query>,<[expected_answer0..n]>")
	clientcount = flag.Int("clients", 1, "Number of client Go Routines to spawn")
	server = flag.String("server", "", "DNS server to target - format '10.1.1.1:53'")
	duration = flag.Int("duration", -1, "How long to run the test for in ms. Negative number will result in one iteration through the test inputs.")
	delay = flag.Int("delay", 100, "Delay between client requests in ms. Use 0 for load test - with CAUTION not to DOS your prod systems!")
	debug = flag.Bool("debug", false, "Enable debug messages")
	jsonout = flag.Bool("json", false, "Ouput in json format")
	validate = flag.Bool("validate", false, "Validate the 1st response matches the expected set of responses.")
	outputinfo = `
	The test output will show raw stats per client and overall, followed by
	summary data.

	Errors: (count) the server did not respond or there was a unexpected error.
	Noanswer: (count) the server response was recieved by there was no answers to the query.
	Success: (count) the server response had answers
	Verified: (count) the server response was validated against the expected result set
	Incorrect: (count) the server response was not found in the expected result set
	QPS: queries per second for the test run
	RTT-min: minimum round trip time observed
	RTT-max: maximum round trip time observed
	RTT-avg: average round trip time observed
	Test duration(ms): overall test duration
	`
)

type lookuprecord struct {
	qtype uint16
	name string
	expects []string
}
func newLookuprecord(qtype uint16, name string, expects []string) *lookuprecord {
	lr := lookuprecord{qtype: qtype, name: name, expects: expects}
	return &lr
}

type teststat struct {
	Errors int
	Noanswer int
	Success int
	Rtttotal int
	Rttmin int
	Rttmax int
	Rttavg int
	Verified int
	Incorrect int
}
func (ts *teststat) setMinMax(rtt int) {
	if rtt < ts.Rttmin || ts.Rttmin < 0 { ts.Rttmin = rtt }
	if rtt > ts.Rttmax { ts.Rttmax = rtt }
}
func (ts *teststat) setAvg() {
	if ts.Success == 0 {
		ts.Rttavg = -1
	} else {
		ts.Rttavg = ts.Rtttotal / ts.Success
	}
}
func newTeststat() *teststat {
	ts := teststat{
		Errors: 0,
		Noanswer: 0,
		Success: 0,
		Rtttotal: 0,
		Rttmin: -1,
		Rttmax: -1,
		Rttavg: -1,
		Verified: 0,
		Incorrect: 0,
	}
	return &ts
}

func dnsClient(wg *sync.WaitGroup, startSignal chan struct{}, stopSignal chan struct{}, statchannel chan teststat, testlist *[]lookuprecord) {
	defer wg.Done()
	c := new(dns.Client)
	stats := newTeststat()
	<- startSignal
	mainloop:
	for {
		for _, q := range *testlist {
			select {
			case <- stopSignal:
				break mainloop
			default:
				m := new(dns.Msg)
				m.SetQuestion(q.name, q.qtype)
				in, rtt, err := c.Exchange(m, *server)
				if err != nil {
					log.WithFields(log.Fields{
						"error": err.Error(),
						"request": m,
						}).Debug("Failure for dns query")
					stats.Errors++
				} else if len(in.Answer) < 1 {
					log.WithFields(log.Fields{
						"response": in,
						"request": m,
						}).Debug("No answers found for query")
					stats.Noanswer++
				} else {
					log.WithFields(log.Fields{
						"answers": in.Answer,
						"rtt": rtt.Milliseconds(),
						"request": m,
						}).Debug("Successful answers for query")
					stats.Success++
					stats.setMinMax(int(rtt.Milliseconds()))
					stats.Rtttotal = stats.Rtttotal + int(rtt.Milliseconds())
					if *validate {
						// check if response is in the expected response list
						var answerValue string
						switch in.Answer[0].(type) {
						case *dns.A:
							answerValue = in.Answer[0].(*dns.A).A.String()
						case *dns.CNAME:
							answerValue = in.Answer[0].(*dns.CNAME).Target
						case *dns.MX:
							answerValue = in.Answer[0].(*dns.MX).Mx 
						}
						gotmatch := false
						for _, exp := range q.expects {
							if strings.EqualFold(answerValue, exp) {
								stats.Verified++
								gotmatch = true
								break
							}
						}
						if !gotmatch { 
							stats.Incorrect++
							log.WithFields(log.Fields{
								"first_answer": answerValue,
								"expected": q.expects,
								"request": m,
								}).Warn("Answer does not match expected response")
						}
					}
				}
				time.Sleep( time.Duration(*delay) * time.Millisecond)
			}
		}
		if *duration < 0 { break mainloop } // run once if duration < 0
	}
	stats.setAvg()
	statchannel <- *stats
}

func aggregateResults(results[]teststat) *teststat {
	agg := newTeststat()
	for _,s := range results {
		agg.Errors = agg.Errors + s.Errors
		agg.Success = agg.Success + s.Success
		agg.Noanswer = agg.Noanswer + s.Noanswer
		agg.Rtttotal =  agg.Rtttotal + s.Rtttotal
		agg.setMinMax(s.Rttmin)
		agg.setMinMax(s.Rttmax)
		agg.Verified = agg.Verified + s.Verified
		agg.Incorrect = agg.Incorrect + s.Incorrect
	}
	agg.setAvg()
	return agg
}

func readInputs(inputfile string) *[]lookuprecord{
	var inputs []lookuprecord
	csvfile, err := os.Open(inputfile)
	if err != nil {
		log.Fatalln("Couldn't open the input csv file", err)
	}
	// handle utf bom at start of file (often from Excel exports)
	sr, _ := utfbom.Skip(csvfile)
	// Parse the file
	r := csv.NewReader(sr)
	r.FieldsPerRecord = -1
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		log.WithFields(log.Fields{
			"type": record[0],
			"question": record[1],
			"expects": record[2:],
			}).Debug("Input record")
		var qtype uint16
		if k, ok := dns.StringToType[strings.ToUpper(record[0])]; ok {
			qtype = k
			inputs = append(inputs, *newLookuprecord(qtype, record[1], record[2:]))
		} else {
			log.WithFields(log.Fields{
				"type": record[0],
				"question": record[1],
				}).Warn("Invalid or unsupported record type - excluding from test")
		}
		
	}
	return &inputs
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s \n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "Output: %s \n", outputinfo)
		fmt.Fprintf(os.Stderr, "Version: %s \n", version)
	}
	flag.Parse()
	if *debug {
		log.SetLevel(log.DebugLevel)
	}
	
	var wg sync.WaitGroup
	var results []teststat
	testlist := readInputs(*inputfile)
	statch := make(chan teststat)
	startSignal := make(chan struct{})
	stopSignal := make(chan struct{})

	// Start clients routines
	for client := 0; client < *clientcount; client++ {
		wg.Add(1)
		go dnsClient(&wg, startSignal, stopSignal, statch, testlist)
		//wg.Wait()
	}
	close(startSignal) // signal start
	start := time.Now()
	if *duration > 0 {
		time.Sleep( time.Duration(*duration) * time.Millisecond)
		close(stopSignal) // signal end
	} 
	// get stats from all clients
	for client := 0; client < *clientcount; client++ {
		results = append(results, <-statch)
	}
	timed := time.Since(start)

	aggResult := aggregateResults(results)
	
	qps := float64((*aggResult).Success)/timed.Seconds()
	
	if *jsonout {
		type Output struct {
			Duration int64
			QPS float64
			Summary teststat
			Detail []teststat
		}
		outdata := Output{	Duration: timed.Milliseconds(),
							QPS: qps,
							Summary: *aggResult, 
							Detail: results}
		data, err := json.Marshal(outdata)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", data)
	} else {
		fmt.Printf("Client Results \n%v\n", results)
		fmt.Printf("Overall Results \n%v\n", *aggResult)
		fmt.Printf(	"--------------------------------------------------------\n"+
					"Result Summary\n"+
					"--------------------------------------------------------\n"+
					"  Errors: %d\n"+
					"  Noanswer: %d\n"+
					"  Success: %d\n"+
					"  Verified: %d\n"+
					"  Incorrect: %d\n"+
					"  QPS: %f\n"+
					"  RTT-min: %d\n"+
					"  RTT-max: %d\n"+
					"  RTT-avg: %d\n"+
					"  Test duration(ms): %d\n"+
					"\n", (*aggResult).Errors,(*aggResult).Noanswer,(*aggResult).Success,
					(*aggResult).Verified, (*aggResult).Incorrect,
					qps, (*aggResult).Rttmin,(*aggResult).Rttmax,(*aggResult).Rttavg, timed.Milliseconds())
	}
}