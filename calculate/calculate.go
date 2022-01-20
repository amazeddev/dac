package calculate

import (
	"dac/execute"
	"dac/parser"
	"fmt"
	"sync"
)

func Find(slice []string, val string) (int) {
	for i, item := range slice {
		if item == val {
			return i
		}
	}
	return -1
}

func CalculateChain(
	chain parser.Chain, 
	i int, 
	ge string, // group element
	workflow parser.Workflow, 
	chain_map map[string]parser.ChainMapElem, 
	import_map map[string]parser.ImportMapElem, 
	no_run_chains []string,
	home string,
	wg *sync.WaitGroup, 
	calcchan chan parser.CalcResults,
) {
	success := true
	status := "completed"
	cr := parser.ImportResp{}

	if chain.Link != "" && Find(no_run_chains, chain.Link) != -1 {
		success = false
		status = "omitted"
	} else {
		targets := []parser.ResultElem{}
	
		ge_slice := []string{}
		if ge != "" {
			ge_slice = append(ge_slice, ge)
		}
	
		// find targets
		if chain.Link != "" {
			linked_chain := workflow.Chains[chain_map[chain.Link].Idx]
	
			if len(chain.Target) == 0 {
				targets = append(targets, linked_chain.Results...)
			} else {
				for _, r := range linked_chain.Results {
					for _, t := range append(chain.Target, ge_slice...) {
						if t == r.Name {
							targets = append(targets, r)
						}
					}
				}
			}
		} else {
			for _, t := range append(chain.Target, ge_slice...) {
				targets = append(targets, parser.ResultElem{Id: import_map[t].Id, Name: t} )
			}
		}
	
		out, err := execute.RunChain(home, parser.CalcChain{Id: chain.Id, Name: chain.Name, Target: targets, Steps: chain.Steps, Link: chain.Link})
		if err != nil {
			fmt.Println("Error: ", err)
		}
	
		cr.ParseImportResp(out)

		if len(cr.Errors) > 0 {
			success = false
			status = "error"
		}
	}

	calcchan <- parser.CalcResults{Chain: chain, Responses: cr.Resp, Errors: cr.Errors, RunType: "", Success: success, Status: status}
}