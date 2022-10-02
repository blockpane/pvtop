package prevotes

import (
	"context"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	staketypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
)

var (
	ChainName string
	VotePower = types.NewInt(0)
	vp        uint64
)

type ValNames struct {
	mux sync.RWMutex

	key    map[string]int // pubkey -> position
	indice map[int]string // index -> moniker
	power  map[int]float64
}

func GetValNames(addr string) *ValNames {
	addr = strings.Replace(addr, "tcp://", "http://", 1)
	httpAddr := strings.TrimRight(addr, "/")
	v := &ValNames{
		key:    make(map[string]int),
		indice: make(map[int]string),
		power:  make(map[int]float64),
	}

	perPage := 100
	page := 1
	index := 0
	more := true

	for more {
		resp, err := http.Get(httpAddr + "/validators?per_page=" + strconv.Itoa(perPage) + "&page=" + strconv.Itoa(page))
		if err != nil {
			log.Fatal(err)
		}
		r, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
		valResp := &rpcValidatorsResp{}
		err = json.Unmarshal(r, valResp)
		if err != nil {
			log.Fatal(err)
		}

		for _, val := range valResp.Result.Validators {
			v.setKey(val.PubKey.Value, index)
			i, _ := strconv.ParseInt(val.VotingPower, 10, 64)
			vp += uint64(i)
			index += 1
		}
		totalVals, _ := strconv.Atoi(valResp.Result.Total)
		if index < totalVals {
			page += 1
		} else {
			more = false
		}
	}

	page = 1
	index = 0
	more = true

	// do it again, but get the % of voting power
	for more {
		resp, err := http.Get(httpAddr + "/validators?per_page=" + strconv.Itoa(perPage) + "&page=" + strconv.Itoa(page))
		if err != nil {
			log.Fatal(err)
		}
		r, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
		valResp := &rpcValidatorsResp{}
		err = json.Unmarshal(r, valResp)
		if err != nil {
			log.Fatal(err)
		}

		for _, val := range valResp.Result.Validators {
			i, _ := strconv.ParseInt(val.VotingPower, 10, 64)
			v.setPower(index, 100*(float64(i)/float64(vp)))
			index += 1
		}
		totalVals, _ := strconv.Atoi(valResp.Result.Total)
		if index < totalVals {
			page += 1
		} else {
			more = false
		}
	}

	client, err := rpchttp.New(addr, "/websocket")
	if err != nil {
		log.Fatal(err)
	}
	var limit uint64 = 100
	var offset uint64 = 0

	for {
		valsQuery := staketypes.QueryValidatorsRequest{
			Status:     "BOND_STATUS_BONDED",
			Pagination: &query.PageRequest{Limit: limit, Offset: offset},
		}
		q, e := valsQuery.Marshal()
		if e != nil {
			log.Fatal(e)
		}
		valsResult, e := client.ABCIQuery(context.Background(), "/cosmos.staking.v1beta1.Query/Validators", q)
		if e != nil {
			log.Fatal(e)
		}
		if len(valsResult.Response.Value) > 0 {
			valsResp := staketypes.QueryValidatorsResponse{}
			e = valsResp.Unmarshal(valsResult.Response.Value)
			if e != nil {
				log.Fatal(e)
			}
			for _, val := range valsResp.Validators {
				annoyed := make(map[string]interface{})
				e = yaml.Unmarshal([]byte(val.String()), &annoyed)
				if e != nil {
					log.Fatal(e)
				}
				i := v.getByKey(annoyed["consensus_pubkey"].(map[string]interface{})["key"].(string))
				v.setIndex(i, strings.TrimSpace(val.Description.Moniker))
			}
			if len(valsResp.Pagination.GetNextKey()) > 0 {
				offset += 1
			} else {
				break
			}
		}
	}

	return v
}

func (v *ValNames) setKey(key string, position int) {
	v.mux.Lock()
	defer v.mux.Unlock()

	v.key[key] = position
}

func (v *ValNames) setIndex(index int, moniker string) {
	v.mux.Lock()
	defer v.mux.Unlock()

	v.indice[index] = moniker
}

func (v *ValNames) setPower(index int, power float64) {
	v.mux.Lock()
	defer v.mux.Unlock()

	v.power[index] = power
}

func (v *ValNames) getByKey(key string) int {
	v.mux.RLock()
	defer v.mux.RUnlock()

	return v.key[key]
}

func (v *ValNames) getByIndex(index int) string {
	v.mux.RLock()
	defer v.mux.RUnlock()

	return v.indice[index]
}

func (v *ValNames) getPower(index int) float64 {
	v.mux.RLock()
	defer v.mux.RUnlock()

	return float64(v.power[index]) / 100
}

func (v *ValNames) GetInfo(index int) string {
	moniker := v.getByIndex(index)
	if len([]byte(moniker)) > 20 {
		moniker = string(append([]byte(moniker[:14]), []byte("...")...))
	}
	if len([]byte(moniker)) > len(moniker) {
		moniker = moniker[:len([]byte(moniker))-len(moniker)]

	}
	return fmt.Sprintf("%-3d %-.2f%%   %-20s ", index+1, v.getPower(index)*100.0, moniker)
}

type rpcValidatorsResp struct {
	Result struct {
		Validators []struct {
			PubKey      struct{ Value string } `json:"pub_key"`
			VotingPower string                 `json:"voting_power"`
		} `json:"validators"`
		Count string `json:"count"`
		Total string `json:"total"`
	} `json:"result"`
}

type status struct {
	Result struct {
		NodeInfo struct {
			Network string `json:"network"`
		} `json:"node_info"`
	} `json:"result"`
}

func GetNetworkName(addr string) (string, error) {
	addr = strings.TrimRight(strings.Replace(addr, "tcp://", "http://", 1), "/")
	resp, err := http.Get(addr + "/status")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	r, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	s := &status{}
	err = json.Unmarshal(r, s)
	if err != nil {
		return "", err
	}
	return s.Result.NodeInfo.Network, nil
}
