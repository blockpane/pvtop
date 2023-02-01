package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/blockpane/pvtop/prevotes"
)

var refreshRate = time.Second

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	var fastPolling bool
	flag.BoolVar(&fastPolling, "fast", false, "fast polling mode, only use if you have a fast connection to the chain")
	flag.Parse()

	if fastPolling {
		refreshRate = 250 * time.Millisecond
	}

	if len(flag.Args()) < 1 {
		printHelp()
		log.Fatal("Exiting")
	}

	rpcHost := flag.Arg(0)
	providerHost := flag.Arg(0)

	// A provider must be specified when targeting a consumer chain
	if len(flag.Args()) == 2 {
		providerHost = flag.Arg(1)
	}

	networkName, err := prevotes.GetNetworkName(rpcHost)
	if err != nil {
		log.Fatal(err)
	}

	// Only the provider host has validator name information
	log.Println("Chain RPC Host: ", rpcHost)
	log.Println("Provider Host: ", providerHost)
	log.Println("Please wait, getting validator information....")
	log.Println("\tIf stalled here:")
	log.Println("\tNormal Chains: Check connectivity to RPC Host")
	log.Println("\tConsumer Chains: Specify and check connectivity to Provider RPC Host")
	printHelp()
	v := prevotes.GetValNames(providerHost)
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
		votes, votePct, commitPct, hrs, dur, propIdx, e := prevotes.GetHeightVoteStep(rpcHost, v)
		if e != nil {
			SummaryChan <- e.Error()
			continue
		}
		if dur < 0 {
			dur = 0
		}
		proposer := "error getting proposer"
		if propIdx >= 0 {
			proposer = v.GetInfo(propIdx)
		}
		SummaryChan <- fmt.Sprintf("height/round/step: %s - v: %.0f%% c: %.0f%% (%v)\n\nProposer:\n(rank/%%/moniker) %s", hrs, votePct*100, commitPct*100, dur, proposer)
		voteChan <- votes
		votePctChan <- votePct
		commitPctChan <- commitPct
	}
}

func printHelp() {
	log.Printf("\n\tSyntax: pvtop <flags> [chainRpcHost] [providerRpcHost]\n" +
		"\tchainRpcHost and providerRpcHost formatted as tcp://127.0.0.0:26657\n" +
		"\tproviderRpcHost only required for consumer chains (typically only cosmoshub)\n\n" +
		"\tflags:\n" +
		"\t\t'-fast' to enable fast polling, at 250ms interval (not recommended)\n")
}
