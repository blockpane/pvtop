package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/blockpane/pvtop/prevotes"
)

const refreshRate = time.Second

func printHelp() {
	log.Printf("\n\tSyntax: pvtop [chainRpcHost] [providerRpcHost]\n\tchainRpcHost and providerRpcHost formatted as tcp://127.0.0.0:26657\n\tproviderRpcHost only required for consumer chains (typically only cosmoshub)")
}
func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	if len(os.Args) < 2 {
		printHelp()
		log.Fatal("Exiting")
	}

	rpcHost := os.Args[1]
	providerHost := os.Args[1]

	// A provider must be specified when targeting a consumer chain
	if len(os.Args) == 3 {
		providerHost = os.Args[2]
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
		votes, votePct, commitPct, hrs, dur, e := prevotes.GetHeightVoteStep(rpcHost, v)
		if e != nil {
			SummaryChan <- e.Error()
			continue
		}
		if dur < 0 {
			dur = 0
		}
		SummaryChan <- fmt.Sprintf("height/round/step: %s - v: %.0f%% c: %.0f%% (%v)\n", hrs, votePct*100, commitPct*100, dur)
		voteChan <- votes
		votePctChan <- votePct
		commitPctChan <- commitPct
	}
}
