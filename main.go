package main

import (
	"fmt"
	"github.com/blockpane/pvtop/prevotes"
	"log"
	"os"
	"time"
)

const refreshRate = time.Second

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	if len(os.Args) < 2 {
		log.Fatal("please provide an rpc endpoint as the only argument")
	}

	parsedAddress, err := prevotes.NewRPCAddress(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	networkName, err := prevotes.GetNetworkName(parsedAddress)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Please wait, getting validator information....")
	v := prevotes.GetValNames(parsedAddress)
	if v == nil {
		log.Fatal("no validators found")
	}

	voteChan := make(chan []prevotes.VoteState)
	pctChan := make(chan float64)
	SummaryChan := make(chan string)

	go prevotes.DrawScreen(networkName, voteChan, pctChan, SummaryChan)

	tick := time.NewTicker(refreshRate)
	for range tick.C {
		votes, pct, hrs, dur, e := prevotes.GetPreVotes(parsedAddress, v)
		if e != nil {
			log.Fatal(e)
		}
		if dur < 0 {
			dur = 0
		}
		SummaryChan <- fmt.Sprintf("height/round/step: %s - pct: %.0f%% (%v)\n", hrs, pct*100, dur)
		voteChan <- votes
		pctChan <- pct
	}
}
