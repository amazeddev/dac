package differ

import (
	"dac/parser"
	"reflect"
)

func SortLinked(groups [][]parser.Chain, groupId int, linked []parser.Chain) [][]parser.Chain {
	newLinked := []parser.Chain{}
	keys := make(map[string]bool)
	groups = append(groups, []parser.Chain{})

	for i := range linked {
		match := parser.Chain{}
		for j := range groups[groupId] {
			if linked[i].Link == groups[groupId][j].Name {
				match = linked[i]
			}
		}
		if match.Name == "" {
			newLinked = append(newLinked, linked[i])
		} else {
			if _, value := keys[match.Name]; !value {
				keys[match.Name] = true
				groups[groupId+1] = append(groups[groupId+1], linked[i])
			}
		}
	}

	if len(newLinked) == 0 {
		return groups
	}

	return SortLinked(groups, groupId+1, newLinked)
}

func FindName(slice []parser.Chain, key string) bool {
	for i := range slice {
    if slice[i].Name == key {
      return true
    }
	}
	return false
}

func GroupChains(chains []parser.Chain) [][]parser.Chain {
	groups := [][]parser.Chain{[]parser.Chain{}}
	linked := []parser.Chain{}

	// find first group - not linked chains
	for i, c := range chains {
		if c.Link == "" || !FindName(chains, c.Link) {
			groups[0] = append(groups[0], c)
		} else {
			linked = append(linked, chains[i])
		}
	}

	if len(linked) > 0 {
		return SortLinked(groups, 0, linked)
	}

	return groups
}

func ConnectedChains(chain parser.Chain, groups [][]parser.Chain) []parser.Chain {
	connected := [][]parser.Chain{[]parser.Chain{}}

	connectedFlag := false

	Groups: for i, group := range groups {
		if !connectedFlag {
			for _, ch := range group {
				if ch.Name == chain.Name {
					connected[0] = append(connected[0], ch)
					connectedFlag = true
					continue Groups
				}
			}
		}

		if connectedFlag {
			linkedGroup := []parser.Chain{}
			for _, ch := range group {
				for _, conn := range connected[len(connected)-1] {
					if conn.Name == ch.Link {
						linkedGroup = append(linkedGroup, ch)
					}
				}
			}

			if len(linkedGroup) == 0 {
				break Groups
			} else {
				connected = append(connected, linkedGroup)
			}
		}


		if i == len(groups)-1 {
			break Groups
		}
	}
	
	// flatten connected nested slice
	flat := []parser.Chain{}
	for _, group := range connected {
		flat = append(flat, group...)
	}
	
	return flat
}

func ConnectedChainsMulti(connected, configChains []parser.Chain ) []parser.Chain {
	affectedChains := []parser.Chain{}
	groups := GroupChains(configChains)
	for _, chain := range connected {
		chainConnected := ConnectedChains(chain, groups)
		for _, cc := range chainConnected {
			if !FindName(affectedChains, cc.Name) {
				affectedChains = append(affectedChains, cc)
			}
		}
	}
	return affectedChains
}

func ChainEqual(old, new parser.Chain) bool {
	if old.Name != new.Name {
		return false
	}
	return true
}


func FindDiffs(old, new []parser.Chain) (map[int]parser.Chain, map[int]parser.Chain, map[int]parser.Chain) {
	deleted := map[int]parser.Chain{}
	common := map[int]parser.Chain{}
	updated := map[int]parser.Chain{}
	created := map[int]parser.Chain{}

	if len(old) == 0 {
		for i, el := range new {
			created[i] = el
		}
		return deleted, updated, created
	}

	OldLoop: for i, oel := range old {
		for j, nel := range new {
			if reflect.DeepEqual(oel, nel) {
				common[j] = oel
				continue OldLoop
			} else if oel.Name == nel.Name {
				updated[j] = nel
				continue OldLoop
			}
		}
		deleted[i] = oel
	}
	for i, nel := range new {
		_, cok := common[i]
		_, uok := updated[i]
		if !cok && !uok {
			created[i] = nel
		}
	}

	return deleted, updated, created
}

func Find(slice []string, val string) (int) {
	for i, item := range slice {
		if item == val {
			return i
		}
	}
	return -1
}

func FindLink(chain parser.Chain, workflow parser.Workflow, chains_names []string, operation_chains []string, storagecols []string) []parser.Chain {
	runChains := []parser.Chain{}
	
	link_chain := workflow.Chains[Find(chains_names, chain.Link)]
	if len(link_chain.Results) == 0 {
		runChains = append([]parser.Chain{link_chain}, runChains...)
	} else {
		for _, res := range link_chain.Results {
			if Find(storagecols, res.Id) == -1 {
				runChains = append([]parser.Chain{link_chain}, runChains...)
				break
			}
		}
	}

	if link_chain.Link != "" && Find(operation_chains, chain.Link) == -1 {
		return append(FindLink(link_chain, workflow, chains_names, operation_chains, storagecols), runChains...)
	} else {
		return runChains
	}
}