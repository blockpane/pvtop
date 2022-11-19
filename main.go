package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/blockpane/pvtop/prevotes"
)

const refreshRate = time.Second

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	if len(os.Args) < 2 {
		log.Fatal("please provide an rpc endpoint as the only argument")
	}

	networkName, err := prevotes.GetNetworkName(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Please wait, getting validator information....")
	v := prevotes.GetValNames(os.Args[1])
	if v == nil {
		log.Fatal("no validators found")
	}

	voteChan := make(chan []prevotes.VoteState)
	votePctChan := make(chan float64)
	commitPctChan := make(chan float64)
	SummaryChan := make(chan string)

	go prevotes.DrawScreen(networkName, voteChan, votePctChan, commitPctChan, SummaryChan)

	tick := time.NewTicker(refreshRate)
	for range tick.C {
		votes, votePct, commitPct, hrs, dur, e := prevotes.GetHeightVoteStep(os.Args[1], v)
		if e != nil {
			SummaryChan <- e.Error()
			continue
		}
		if dur < 0 {
			dur = 0
		}
		SummaryChan <- fmt.Sprintf("height/round/step: %s - v: %.2f%% c: %.2f%% (%v)\n", hrs, votePct*100, commitPct*100, dur)
		voteChan <- votes
		votePctChan <- votePct
		commitPctChan <- commitPct
	}
}
