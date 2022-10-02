package prevotes

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Hvs struct {
	Prevotes         []string `json:"prevotes"`
	PreVotesBitArray string   `json:"prevotes_bit_array"`
}

type conState struct {
	Result struct {
		RoundState struct {
			HRS            string    `json:"height/round/step"`
			HeightVoteStep []Hvs     `json:"height_vote_set"`
			StartTime      time.Time `json:"start_time"`
		} `json:"round_state"`
	} `json:"result"`
}

func (cs *conState) getRound() (int, error) {
	round := strings.Split(cs.Result.RoundState.HRS, "/")
	if len(round) < 2 {
		return 0, errors.New("invalid round")
	}
	r, err := strconv.Atoi(round[1])
	if err != nil {
		return 0, err
	}
	return r, nil
}

func (cs *conState) getPercent(round int) (float64, error) {
	bitArray := strings.Split(cs.Result.RoundState.HeightVoteStep[round].PreVotesBitArray, " ")
	if len(bitArray) < 3 {
		return 0, errors.New("invalid bit array")
	}
	percent, err := strconv.ParseFloat(bitArray[len(bitArray)-1], 64)
	if err != nil {
		return 0, err
	}
	return percent, nil
}

type VoteState struct {
	Description string
	Voted       bool
}

func GetPreVotes(url string, names *ValNames) (votes []VoteState, percent float64, hrs string, dur time.Duration, err error) {
	votes = make([]VoteState, 0)
	url = strings.TrimRight(strings.ReplaceAll(url, "tcp://", "http://"), "/")
	resp, err := http.Get(url + "/consensus_state")
	if err != nil {
		return nil, 0, "", 0, err
	}
	defer resp.Body.Close()
	r, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, "", 0, err
	}
	state := &conState{}
	err = json.Unmarshal(r, state)
	if err != nil {
		return nil, 0, "", 0, err
	}
	round, err := state.getRound()
	if err != nil {
		return nil, 0, "", 0, err
	}
	for i := range state.Result.RoundState.HeightVoteStep[round].Prevotes {
		vote := state.Result.RoundState.HeightVoteStep[round].Prevotes[i]
		voted := false
		if vote != "nil-Vote" {
			voted = true
		}
		votes = append(votes, VoteState{
			Description: names.GetInfo(i),
			Voted:       voted,
		})
	}
	percent, err = state.getPercent(round)
	if err != nil {
		return nil, 0, "", 0, err
	}
	dur = time.Now().UTC().Sub(state.Result.RoundState.StartTime)
	return votes, percent, state.Result.RoundState.HRS, dur, nil
}
