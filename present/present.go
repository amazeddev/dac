package present

import (
	"dac/parser"
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

func PrintLogo() {
	fmt.Println(string("\033[32m"), "    _\n  __| | __ _  ___\n / _` |/ _` |/ __|\n| (_| | (_| | (__\n \\__,_|\\__,_|\\___|", string("\033[0m"))
	fmt.Println(string("\033[32m"), "data aplications configurator", string("\033[0m"))
	fmt.Println()
}

func PrintCheck(val bool) string {
	if val {
		return fmt.Sprint(string("\033[32m"), "\u2714", string("\033[0m"))
	} else {
		return fmt.Sprint(string("\033[31m"), "\u2718", string("\033[0m"))
	}
}

func PrintChain(chain parser.Chain, t string, verbose bool) {
	var color string
	colorReset := string("\033[0m")

	switch t {
	case "delete":
		color = string("\033[31m")
	case "update":
		color = string("\033[33m")
	case "create":
		color = string("\033[32m")
	default:
		color = string("\033[37m")
	}
	fmt.Println()
	ymlBytes, _ := yaml.Marshal(chain)
	ymlSlice := strings.Split(string(ymlBytes), "\n")
	linePerChain := 1
	if verbose {
		linePerChain = len(ymlSlice) - 1
	}
	for i := range ymlSlice[:linePerChain] {
		line := ymlSlice[i]
		if !verbose {
			line = ymlSlice[i][5:]
		}
		fmt.Printf(" %v %v %v\n", color, line, colorReset)
	}
}

func PrintResult(chain parser.Chain, resps []parser.ResultElem, errors []parser.ErrorElem, runType string, status string ) {
	var color string
	colorReset := string("\033[0m")
	switch runType {
	case "delete":
		color = string("\033[31m")
	case "update":
		color = string("\033[33m")
	case "create":
		color = string("\033[32m")
	default:
		color = string("\033[37m")
	}
	if status == "omitted" {
		color = string("\033[37m")
		runType = fmt.Sprintf("%s - omitted", runType)
	}
	
	fmt.Printf("    chain: %20s;\ttype: %s%s%s\n", chain.Name, color, runType, colorReset)
	fmt.Println("    steps:")
	for _, s := range chain.Steps {
		fmt.Printf("      \u2022 %v\n", s.Function)
		if len(s.Args) > 0{
			for k, v := range s.Args {
				fmt.Printf("\t  %v: %v\n", k, v)
			}
		}
	}

	if len(resps) > 0 {
		if runType == "delete" {
			fmt.Println("    deleted columns:")
		} else {
			fmt.Println("    results:")
		}
		for _, r := range resps {
			fmt.Printf("      \u2022 %v (%+v)\n", r.Name, r.Id)
		}
	}
	if status == "error" {
		fmt.Println("\n   \u2757 error ocured:")
	} else if status == "omitted" {
		fmt.Println("\n   \u2755 chain omitted\n      one chain in workflow might fail, causing dependent on it to be omitted...")

	}
	if len(errors) > 0 {
		for _, err := range errors {
			fmt.Printf("      \u2022 %v \n", err.Error)
		}
	}
	fmt.Println()
}

func PrintHelp() {
	PrintLogo()
	
	fmt.Println("\npossible commands:")
	fmt.Println("  \u2022 init      # initialize project")
	fmt.Println("  \u2022 validate  # validate configuration")
	fmt.Println("  \u2022 compare   # compare current configuration with calculated resources")
	fmt.Println("  \u2022 run       # run current configuration")
	fmt.Println("  \u2022 stop      # stop interactive mode")
	fmt.Println("\nfor more informations about command run 'dac *command* -h'")
}