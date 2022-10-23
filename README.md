# pvtop

A simple utility for watching pre-vote status on Tendermint chains. It will print out the current pre-vote status for each validator in the validator set. Useful for watching pre-votes during an upgrade or other network event causing a slowdown.

## Usage

```
pvtop tcp://localhost:26657
```

## Example

![example](img/pvtop.svg)

## Install

```
git clone https://github.com/blockpane/pvtop
cd pvtop
go install ./...
```
