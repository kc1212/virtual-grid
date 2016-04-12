package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/rpc"
	"os"
	"time"
)

import "github.com/kc1212/virtual-grid/model"

// TODO add interactive option
func main() {
	cli()
}

func cli() {
	addr := flag.String("addr", "localhost:3000", "address:port of the grid scheduler")
	jobsCount := flag.Int("count", 1, "the number of jobs to add")
	duration := flag.Int64("duration", 0, "the duration for the jobs (default is a random value)")
	nodeType := flag.String("type", "gs", "add job on \"gs\" or \"rm\"")
	flag.Parse()

	if *nodeType != "gs" && *nodeType != "rm" {
		flag.PrintDefaults()
		return
	}

	rand.Seed(time.Now().UTC().UnixNano())
	jobs := make([]model.Job, *jobsCount)
	for i := range jobs {
		jobs[i].ID = rand.Int63()
		if *duration == 0 {
			jobs[i].Duration = rand.Int63n(10) + 1
		} else {
			jobs[i].Duration = *duration
		}
	}

	reply := -1
	remote, e := rpc.DialHTTP("tcp", *addr)
	if e != nil {
		log.Printf("Node %v is not online, make sure to use the correct address?\n", *addr)
		return
	}

	if *nodeType == "gs" {
		if e := remote.Call("GridSdr.AddJobsViaUser", &jobs, &reply); e != nil {
			log.Printf("Remote call GridSdr.AddJobsViaUser failed on %v, %v\n", *addr, e.Error())
		}
	} else if *nodeType == "rm" {
		if e := remote.Call("ResMan.AddJobsViaUser", &jobs, &reply); e != nil {
			log.Printf("Remote call ResMan.AddJobsViaUser failed on %v, %v\n", *addr, e.Error())
		}
	}
}

func interactive() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Println(scanner.Text()) // Println will add back the final '\n'
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
}
