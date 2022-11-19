package prevotes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Hvs struct {
	Prevotes           []string `json:"prevotes"`
	Precommits         []string `json:"precommits"`
	PreVotesBitArray   string   `json:"prevotes_bit_array"`
	PreCommitsBitArray string   `json:"precommits_bit_array"`
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

func getPercent(equation string) (float64, error) {
	// equation example: 421849654/633993401
	if len(equation) < 2 {
		return 0, errors.New("invalid equation")
	}

	var dividend, divisor int64
	parsed, err := fmt.Sscanf(equation, "%d/%d", &dividend, &divisor)
	if err != nil {
		return 0, err
	}

	if parsed != 2 {
		return 0, errors.New("invalid equation")
	}

	return float64(dividend) / float64(divisor), nil
}

func (cs *conState) getVotePercent(round int) (float64, error) {
	bitArray := strings.Split(cs.Result.RoundState.HeightVoteStep[round].PreVotesBitArray, " ")
	if len(bitArray) < 3 {
		return 0, errors.New("invalid bit array")
	}
	return getPercent(bitArray[len(bitArray)-3])
}

func (cs *conState) getCommitPercent(round int) (float64, error) {
	bitArray := strings.Split(cs.Result.RoundState.HeightVoteStep[round].PreCommitsBitArray, " ")
	if len(bitArray) < 3 {
		return 0, errors.New("invalid bit array")
	}
	return getPercent(bitArray[len(bitArray)-3])
}

type VoteState struct {
	Description string
	Voted       bool
	VotedZeroes bool
	Committed   bool
}

func GetHeightVoteStep(url string, names *ValNames) (votes []VoteState, votePercent, commitPercent float64, hrs string, dur time.Duration, err error) {
	votes = make([]VoteState, 0)
	url = strings.TrimRight(strings.ReplaceAll(url, "tcp://", "http://"), "/")
	resp, err := http.Get(url + "/consensus_state")
	if err != nil {
		return nil, 0, 0, "", 0, err
	}
	defer resp.Body.Close()
	r, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, 0, "", 0, err
	}
	state := &conState{}
	err = json.Unmarshal(r, state)
	if err != nil {
		return nil, 0, 0, "", 0, err
	}
	round, err := state.getRound()
	if err != nil {
		return nil, 0, 0, "", 0, err
	}
	for i := range state.Result.RoundState.HeightVoteStep[round].Prevotes {
		vote := state.Result.RoundState.HeightVoteStep[round].Prevotes[i]
		voted := false
		votedZeroes := false
		if vote != "nil-Vote" {
			voted = true
		}

		if strings.Contains(vote, "SIGNED_MSG_TYPE_PREVOTE(Prevote) 000000000000") {
			votedZeroes = true
		}

		votes = append(votes, VoteState{
			Description: names.GetInfo(i),
			Voted:       voted,
			VotedZeroes: votedZeroes,
		})
	}
	for i := range state.Result.RoundState.HeightVoteStep[round].Precommits {
		commit := state.Result.RoundState.HeightVoteStep[round].Precommits[i]
		committed := false
		if commit != "nil-Vote" {
			committed = true
		}
		votes[i].Committed = committed
	}
	votePercent, err = state.getVotePercent(round)
	if err != nil {
		return nil, 0, 0, "", 0, err
	}
	commitPercent, err = state.getCommitPercent(round)
	if err != nil {
		return nil, 0, 0, "", 0, err
	}
	dur = time.Now().UTC().Sub(state.Result.RoundState.StartTime)
	return votes, votePercent, commitPercent, state.Result.RoundState.HRS, dur, nil
}
