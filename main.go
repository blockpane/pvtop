package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/blockpane/pvtop/prevotes"
)

const refreshRate = time.Second

var lastValUpdate = time.Now()
var reqValUpdate = make(chan bool)

func maintainValNames(v *prevotes.ValNames) {
	for {
		v.Update()
		lastValUpdate = time.Now()
		<-reqValUpdate
	}
}

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
	v := prevotes.NewValNamesWithAddr(os.Args[1])
	go maintainValNames(v)

	voteChan := make(chan []prevotes.VoteState)
	votePctChan := make(chan float64)
	commitPctChan := make(chan float64)
	SummaryChan := make(chan string)

	go prevotes.DrawScreen(networkName, voteChan, votePctChan, commitPctChan, SummaryChan)

	tick := time.NewTicker(refreshRate)
	lastValUpdateHeight := 0
	for range tick.C {
		votes, votePct, commitPct, hrs, dur, e := prevotes.GetHeightVoteStep(os.Args[1], v)
		if e != nil {
			SummaryChan <- e.Error()
			continue
		}
		if dur < 0 {
			dur = 0
		}
		SummaryChan <- fmt.Sprintf("Height/Round/Step: %s - v: %.0f%% c: %.0f%% (%v)\nVote Power Updated @h=%d (%s)",
			hrs, votePct*100, commitPct*100, dur,
			lastValUpdateHeight, time.Now().Sub(lastValUpdate))
		voteChan <- votes
		votePctChan <- votePct
		commitPctChan <- commitPct

		currentHeight, err := strconv.Atoi(strings.Split(hrs, "/")[0])
		if err == nil {
			// Update every 10 blocks, or immediately for new chains
			if (lastValUpdateHeight+10 < currentHeight) || (currentHeight < 3) {
				reqValUpdate <- true
				lastValUpdateHeight = currentHeight
			}
		}

	}
}
