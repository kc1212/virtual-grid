package model

import (
	"log"
	"sync"
	"time"
)

import "github.com/kc1212/virtual-grid/common"
import "github.com/kc1212/virtual-grid/discosrv"

type ResMan struct {
	common.Node
	n             int // number of workers
	gsNodes       *common.SyncedSet
	tasksChan     chan WorkerTask
	completedChan chan int64
	capReq        chan int
	capResp       chan int
	discosrvAddr  string
}

func InitResMan(n int, id int, addr string, dsAddr string) ResMan {
	return ResMan{
		common.Node{ID: id, Addr: addr, Type: common.RMNode},
		n,
		&common.SyncedSet{S: make(map[string]common.IntClient)},
		make(chan WorkerTask, 1000),
		make(chan int64),
		make(chan int),
		make(chan int),
		dsAddr}
}

// RunResMan is the main function, it starts all its services.
func (rm *ResMan) Run() {
	reply, e := discosrv.ImAliveProbe(rm.Addr, common.RMNode, rm.discosrvAddr)
	if e != nil {
		log.Panicf("Discosrv on %v not online\n", rm.discosrvAddr)
	}
	rm.notifyAndPopulateGSs(reply.GSs)

	go discosrv.ImAlivePoll(rm.Addr, common.RMNode, rm.discosrvAddr)
	go common.RunRPC(rm, rm.Addr)
	go runWorkers(rm.n, rm.tasksChan, rm.capReq, rm.capResp, rm.completedChan)
	rm.handleCompletionMsg()
}

// AddJob RPC call
func (rm *ResMan) AddJob(jobs *[]Job, reply *int) error {
	log.Printf("Jobs received %v\n", *jobs)

	// make a channel of jobs, and then schedule them
	for _, j := range *jobs {
		// in theory the task can be arbitrary, here we just run Sleep
		task := func() (interface{}, error) {
			time.Sleep(time.Duration(j.Duration) * time.Second)
			return 0, nil
		}
		rm.tasksChan <- WorkerTask{task, j.ID}
	}
	*reply = 0
	return nil
}

// RecvMsg PRC call
func (rm *ResMan) RecvMsg(args *RPCArgs, reply *int) error {
	// log.Printf("Msg received %v\n", *args)
	*reply = -1
	if args.Type == common.RMUpMsg {
		*reply = rm.ID
		rm.gsNodes.SetInt(args.Addr, int64(args.ID))

	} else if args.Type == common.GetCapacityMsg {
		*reply = rm.computeCapacity()

	} else {
		log.Panic("Invalid message!", args)
	}
	return nil
}

func (rm *ResMan) computeCapacity() int {
	rm.capReq <- 0
	cap := <-rm.capResp
	return cap
}

func (rm *ResMan) notifyAndPopulateGSs(nodes []string) {
	// NOTE: does RM doesn't use a clock, hence the zero
	arg := RPCArgs{rm.ID, rm.Addr, common.RMUpMsg, 0}
	for _, node := range nodes {
		id, e := rpcSendMsgToGS(node, &arg)
		if e == nil {
			rm.gsNodes.SetInt(node, int64(id))
		}
	}
}

func (rm *ResMan) handleCompletionMsg() {
	ids := make([]int64, 0)
	mutex := sync.Mutex{}

	// update the ids array when something arrives in completedChan
	go func() {
		for {
			for id := range rm.completedChan {
				mutex.Lock()
				ids = append(ids, id)
				mutex.Unlock()
			}
		}
	}()

	// send the ids to GS every 100ms
	for {
		time.Sleep(100 * time.Millisecond)
		if len(ids) == 0 {
			continue
		}

		mutex.Lock()
		// NOTE: better to randomly choose a GS
		for k := range rm.gsNodes.GetAll() {
			_, e := rpcSyncCompletedJobs(k, &ids)
			if e == nil {
				break
			}
		}
		ids = make([]int64, 0)
		mutex.Unlock()
	}
}
