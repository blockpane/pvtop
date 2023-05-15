package prevotes

import (
	"fmt"
	"log"
	"os"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

func DrawScreen(network string, voteChan chan []VoteState, votePctChan, commitPctChan chan float64, summaryChan chan string) {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	votePctGauge := widgets.NewGauge()
	commitPctGauge := widgets.NewGauge()

	p := widgets.NewParagraph()
	p.Title = network

	lists := make([]*widgets.List, 3)
	for i := range lists {
		lists[i] = widgets.NewList()
		lists[i].Border = false
	}
	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	grid.Set(
		ui.NewRow(0.1,
			ui.NewCol(1.0/3, p),
			ui.NewCol(1.0/3, votePctGauge),
			ui.NewCol(1.0/3, commitPctGauge),
		),
		ui.NewRow(0.9,
			ui.NewCol(.9/3, lists[0]),
			ui.NewCol(.9/3, lists[1]),
			ui.NewCol(1.2/3, lists[2]),
		),
	)
	ui.Render(grid)

	refresh := false
	tick := time.NewTicker(100 * time.Millisecond)
	uiEvents := ui.PollEvents()

	for {
		select {

		case <-tick.C:
			if !refresh {
				continue
			}
			refresh = false
			ui.Render(grid)

		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				ui.Clear()
				ui.Close()
				os.Exit(0)
			case "j", "<Down>":
				for _, list := range lists {
					if len(list.Rows) > 0 {
						list.ScrollBottom()
					}
				}
			case "k", "<Up>":
				for _, list := range lists {
					if len(list.Rows) > 0 {
						list.ScrollTop()
					}
				}
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				grid.SetRect(0, 0, payload.Width, payload.Height)
				ui.Clear()
				ui.Render(grid)
			}

		case votes := <-voteChan:
			refresh = true
			split, columns, rows := splitVotes(votes)
			for i := 0; i < columns; i++ {
				lists[i].Rows = make([]string, rows)
				for j, voter := range split[i] {
					vmissing := "❌"
					if voter.Voted {
						vmissing = "✅"
					}
					if voter.VotedZeroes {
						vmissing = "🤷"
					}
					cmissing := "❌"
					if voter.Committed {
						cmissing = "✅"
					}
					lists[i].Rows[j] = fmt.Sprintf("%-3s %-3s %s", vmissing, cmissing, voter.Description)
				}
			}

		case pct := <-votePctChan:
			refresh = true
			votePctGauge.Percent = int(pct * 100)

		case pct := <-commitPctChan:
			refresh = true
			commitPctGauge.Percent = int(pct * 100)

		case summary := <-summaryChan:
			refresh = true
			p.Text = summary

		}
	}
}

func splitVotes(votes []VoteState) ([][]VoteState, int, int) {
	split := make([][]VoteState, 0)
	var columns int
	var rows int = 50

	switch {
	case len(votes) < 50:
		columns = 1
		split = append(split, votes)
	case len(votes) < 100:
		columns = 2
		split = append(split, votes[:50])
		split = append(split, votes[50:])
	default:
		columns = 3
		rows = (len(votes) + columns - 1) / 3
		split = append(split, votes[:rows])
		split = append(split, votes[rows:rows*2])
		split = append(split, votes[rows*2:])
	}
	return split, columns, rows
}
