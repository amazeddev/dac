package main

import (
	"dac/client"
	"dac/differ"
	"dac/execute"
	"dac/interact"
	"dac/parser"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

type GetArgs struct {
	Key  string
}

type ColObj struct {
	Name string
	Type string
	Data []interface{}
	Pres []int
}


type DataArgs struct {
	Name string						`json:"name,omitempty"`
	Key string						`json:"key,omitempty"`
	Tabname string				`json:"tabname,omitempty"`
	Results []string			`json:"results,omitempty"`
	Columns []string			`json:"columns,omitempty"`
	Path string						`json:"path,omitempty"`
}

func MapName(chains []parser.Chain) {
	for i := range chains {
		fmt.Printf("%v ", chains[i].Name)
	}
	fmt.Println()
}

type ChainValiObj struct {
	T     string
	Chain parser.Chain
}

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

func PrintResult(chain parser.Chain, cr parser.ImportResp, runType string ) {
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
	fmt.Println()
	fmt.Printf("  chain: %20s;\ttype: %s%s%s\n", chain.Name, color, runType, colorReset)
	fmt.Println("  steps:")
	for _, s := range chain.Steps {
		fmt.Printf("    \u2022 %v\n", s.Function)
		if len(s.Args) > 0{
			for k, v := range s.Args {
				fmt.Printf("\t%v: %v\n", k, v)
			}
		}
	}

	if len(cr.Resp) > 0 {
		if runType == "delete" {
			fmt.Println("  deleted columns:")
		} else {
			fmt.Println("  result columns:")
		}
		for _, r := range cr.Resp {
			fmt.Printf("    \u2022 %v (%+v)\n", r.Name, r.Id)
		}
	}
	fmt.Println()
}

func PrintHelp() {
	fmt.Println("DAC package help")
	fmt.Println("\npossible commands:")
	fmt.Println("  \u2022 init      # initialize project")
	fmt.Println("  \u2022 validate  # validate current configuration")
	fmt.Println("  \u2022 run       # run current configuration")
	fmt.Println("  \u2022 stop      # stop interactive mode")
	fmt.Println("\nfor more informations about command run 'dac *command* -h'")
}

func Find(slice []string, val string) (int) {
    for i, item := range slice {
        if item == val {
            return i
        }
    }
    return -1
}

func main() {

	help := flag.Bool("h", false, "current environment")
	flag.Parse()
	if *help {
		PrintHelp()
	}

	commands := []string{"init", "list", "import", "validate", "run", "check", "stop", "clear"}
	if Find(commands, os.Args[1]) == -1 {
		fmt.Printf("dac: '%s' is not a dac command; see 'dac -h' for help\n", os.Args[1])
	}

	initCmd := flag.NewFlagSet("init", flag.ExitOnError)

	valCmd := flag.NewFlagSet("validate", flag.ExitOnError)
	valWide := valCmd.Bool("v", false, "# verbose, wheather to show full chain data")
	valAll := valCmd.Bool("a", false, "# all, wheather to show all chain, also not changed ones")

	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	runIntact := runCmd.Bool("i", false, "# interactive mode, not clear storage after run")
	runForce := runCmd.Bool("f", false, "# force mode, ignore previous calculations & run all")

	checkCmd := flag.NewFlagSet("check", flag.ExitOnError)
	checkKey := checkCmd.String("k", "", "# key, stored data table name")

	home, _ := os.UserHomeDir()

	current, _ := os.Getwd()

	switch os.Args[1] {

	case "init":
		initCmd.Parse(os.Args[2:])
		PrintLogo()

		// check setup
		_, err := os.Stat(fmt.Sprintf("%v/%v", home, ".dac"))
		if os.IsNotExist(err) {
			fmt.Println("DAC support directory ~/.dac does not exist.\nProcessing first run steps:")

			kvstorageFileUrl := "https://github.com/amazeddev/kvstorage/releases/download/v0.0.6/kvstorage_0.0.6_linux_amd64.tar.gz"
			_, err := execute.DownloadExtract(kvstorageFileUrl, home)
			if err != nil {
				fmt.Println("Error: ", err)
			}
			fmt.Println("\t- dwonloaded kvstorage")

			libFileUrl := "https://github.com/amazeddev/dac_helpers/releases/download/v0.0.7/release.tar.gz"
			_, err = execute.DownloadExtract(libFileUrl, home)
			if err != nil {
				fmt.Println("Error: ", err)
			}
			fmt.Println("\t- dwonloaded python libs")

			time.Sleep(1 * time.Second)

			_, err = execute.RunConfig(home)
			if err != nil {
				fmt.Println("Error: ", err)
			}
			fmt.Println("\t- initialized python virtualenv")
		}

		dirName := interact.StringPrompt("project directory: ")

		path, _ := os.Getwd()
		v := strings.Split(path, "/")
		projName := dirName

		if dirName == "" {
			dirName = "."
			projName = v[len(v)-1]
		} else {
			path = fmt.Sprintf("%s/%s", path, projName)
		}

		_ = os.MkdirAll(fmt.Sprintf("%v/%v", dirName, ".dac/config"), 0755)
		_ = os.MkdirAll(fmt.Sprintf("%v/%v", dirName, ".dac/data"), 0755)
		_ = os.Mkdir(fmt.Sprintf("%v/%v", dirName, "functions"), 0755)
		_ = os.MkdirAll(fmt.Sprintf("%v/%v", dirName, "modules"), 0755)

		ok := interact.ConfirmPrompt("would like to specify import now?")
		configImport := parser.Import{}

		if ok {
			dataSrcType := interact.SelectPrompt("data source type:", []string{"csv", "SQL"})
			dataSrcPath := interact.StringPrompt("data source path: ")

			configImport.Type = dataSrcType
			configImport.Dataset = dataSrcPath
		}

		baseConfig := parser.Config{
			Name:    projName,
			Engine:  "python",
			Import:  configImport,
			Workflows: []parser.Workflow{{Name: "base", Chains: []parser.Chain{}}},
		}

		err = parser.WriteConfig(baseConfig, fmt.Sprintf("%v/%v", dirName, "main.yml"))
		if err != nil {
			fmt.Println("Error: ", err)
		}

		fmt.Printf("\ncreated project: %v\nin directory:\t %v\n\n", projName, path)

	case "list":
		var kvs client.Rpc_client
		err := kvs.Connect()
		if err != nil {
			fmt.Println("\"list\" could be only used egen engine is initialized!")
		}

		reply, err := kvs.List(struct{}{})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%+v\n", reply)
		fmt.Printf("\n%v columns stored\n", len(reply))

	case "import":

		// TODO check if file exist & add support for SQL

		config, err := parser.Parse() // parsed raw config
		if err != nil {
			log.Fatal(err)
		}
		filepath, _ := filepath.Abs(fmt.Sprintf("%s/%s", current, config.Import.Dataset))
		
		out, err := execute.RunData(home, "imprt", execute.DataArgs{Name: config.Name, Path: filepath})
		if err != nil {
			fmt.Println("Error: ", err)
		}
		fmt.Println(string(out))
		out, err = execute.RunData(home, "check", execute.DataArgs{Name: config.Name, Key: "import"})
		if err != nil {
			fmt.Println("Error: ", err)
		}
		fmt.Printf("\nsuccesfuly imported dataset:\n\n")
		fmt.Println(string(out))

	case "validate":
		valCmd.Parse(os.Args[2:])

		config, err := parser.Parse() // parsed raw config
		if err != nil {
			log.Fatal(err)
		}
		exist_config := []parser.Chain{}
		if _, err := os.Stat(".dac/config/.build.yml"); err == nil {
			parsed, _ := parser.GetConfig(".dac/config/.build.yml")
			exist_config = parsed.Workflows[0].Chains
		}

		deleted, updated, created := differ.FindDiffs(exist_config, config.Workflows[0].Chains)

		validateChains := make([]ChainValiObj, len(config.Workflows[0].Chains))
		for i, e := range config.Workflows[0].Chains {
			validateChains[i] = ChainValiObj{"exist", e}
		}
		for i, e := range created {
			validateChains[i] = ChainValiObj{"create", e}
		}
		for i, e := range updated {
			validateChains[i] = ChainValiObj{"update", e}
		}
		for i, e := range deleted {
			validateChains = append(validateChains[:i], append([]ChainValiObj{{"delete", e}}, validateChains[i:]...)...)
		}

		for _, e := range validateChains {
			if !*valAll && e.T == "exist" {
				continue
			}
			PrintChain(e.Chain, e.T, *valWide)
		}

		fmt.Printf("\ncreated: %v, updated: %v, deleted: %v (of %v chains total)\n", len(created), len(updated), len(deleted), len(config.Workflows[0].Chains))

	case "run":

		runCmd.Parse(os.Args[2:])
		config, err := parser.Parse() // parsed raw config
		if err != nil {
			log.Fatal(err)
		}

		var kvs client.Rpc_client
		err = kvs.Connect()
		if err != nil {
			fmt.Print("(re)initializing project engine... ")
			cmd := exec.Command(fmt.Sprintf("%v/%v", home, ".dac/kvstorage"))
			cmd.Env = os.Environ()
			cmd.Stdin = nil
			cmd.Stdout = nil
			cmd.Stderr = nil
			cmd.ExtraFiles = nil
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Setsid: true,
			}

			if err := cmd.Start(); err != nil {
				panic(err)
			}

			time.Sleep(2 * time.Second)

			err = kvs.Connect()
			if err != nil {
				panic(err)
			}

			_, err := kvs.SetPid(client.PidArgs{Name: config.Name, Pid: fmt.Sprint(cmd.Process.Pid)})
			if err != nil {
				panic(err)
			}
			fmt.Printf("DONE!\n")
		}

		var conf parser.Config // empty object for wide config

		var deleted map[int]parser.Chain
		var updated map[int]parser.Chain
		var created map[int]parser.Chain

		exist_chains := []parser.Chain{}

		exist := false
		storagecols, err := kvs.List(struct{}{})
		if err != nil {
			log.Fatal(err)
		}
		
		fmt.Println("\nrun modes: ")
		fmt.Println(" \u2022 interactive: ", PrintCheck(*runIntact))
		fmt.Println(" \u2022 force:       ", PrintCheck(*runForce))
		
		if _, err := os.Stat(".dac/config/.build.yml"); err == nil && !*runForce {
			exist = true

			conf, _ = parser.GetConfig(".dac/config/.build.wide.yml")
			old_config, _ := parser.GetConfig(".dac/config/.build.yml")
			exist_chains = old_config.Workflows[0].Chains

			cols := map[string]string{}
			for _, e := range conf.Import.Columns {
				cols[e.Id] = e.Name
			}

		} else if errors.Is(err, os.ErrNotExist) || *runForce {
			conf.Name = config.Name
			conf.Engine = config.Engine
			conf.Import.Dataset = config.Import.Dataset

			for i, w := range config.Workflows {
				conf.Workflows = append(conf.Workflows, parser.Workflow{})
				conf.Workflows[i].Name = w.Name
				conf.Workflows[i].Chains = append(conf.Workflows[i].Chains, w.Chains...)
			}

		} else {
			log.Fatal(err)
		}

		deleted, updated, created = differ.FindDiffs(exist_chains, config.Workflows[0].Chains)

		oper_chains_names := []string{}
		for _, chain := range created {
			oper_chains_names = append(oper_chains_names, chain.Name) 
		}
		for _, chain := range updated {
			oper_chains_names = append(oper_chains_names, chain.Name) 
		}

		// *handle DELETED*
		delChains := []parser.Chain{}

		for ind := range deleted {
			delChains = append(delChains, conf.Workflows[0].Chains[ind])
		}

		delAffected := differ.ConnectedChainsMulti(delChains, conf.Workflows[0].Chains)

		if len(delAffected) > 0 {
			fmt.Println("\ndeleted chains: ")
		}
		cols_to_delete := []string{}
		for _, ch := range delAffected {
			for _, e := range ch.Results {
				args := client.DelArgs{Key: e.Id}
				_, err := kvs.Delete(args)
				if err != nil {
					panic(err)
				}
				if Find(cols_to_delete, e.Name) == -1 {
					cols_to_delete = append(cols_to_delete, e.Name)
				}
			}
			PrintResult(ch, parser.ImportResp{Resp: ch.Results}, "delete")
		}

		delete_keys := []int{}
		for k, _ := range deleted {
			delete_keys = append(delete_keys, k)
		} 

		if len(delete_keys) > 0 {
			filtered_chains := conf.Workflows[0].Chains[:0]
			mainloop: for _, chain := range conf.Workflows[0].Chains {
				for _, dch := range deleted {
					if dch.Name == chain.Name {
						continue mainloop
					}
				}
				filtered_chains = append(filtered_chains, chain)
			}
	
			conf.Workflows[0].Chains = filtered_chains
		}

		if len(cols_to_delete) > 0 || *runForce {
			_, err = execute.RunData(home, "remove", execute.DataArgs{Name: config.Name, Columns: cols_to_delete, Key: conf.Workflows[0].Name})
			if err != nil {
				fmt.Println("Error: ", err)
			}
		}

		if *runForce {
			storedcol, err := kvs.List(struct{}{})
			if err != nil {
				log.Fatal(err)
			}

			for _, col := range storedcol {
				args := client.DelArgs{Key: col}
				_, err := kvs.Delete(args)
				if err != nil {
					panic(err)
				}
			}
		}

		// *handle UPDATED*
		updChains := []parser.Chain{}

		for ind := range updated {
			updChains = append(updChains, updated[ind])
		}

		updAffected := differ.ConnectedChainsMulti(updChains, config.Workflows[0].Chains)

		for i, ch := range updAffected {
			var ind int
		confloop:
			for i, c := range conf.Workflows[0].Chains {
				if c.Name == ch.Name {
					ind = i
					for _, e := range c.Results {
						args := client.DelArgs{Key: e.Id}
						_, err := kvs.Delete(args)
						if err != nil {
							panic(err)
						}
					}
					break confloop
				}
			}

			new_chain := parser.Chain{Name: ch.Name, Link: ch.Link, Target: ch.Target}
			if conf.Workflows[0].Chains[ind].Id == "" {
				new_chain.Id = strings.Replace(uuid.New().String(), "-", "", -1)
			} else {
				new_chain.Id = conf.Workflows[0].Chains[ind].Id
			}
			new_chain.Steps = append(new_chain.Steps, ch.Steps...)


			conf.Workflows[0].Chains[ind] = new_chain
			updAffected[i] = new_chain
		}

		// * handle CREATED *
		creChains := []parser.Chain{}

		for ind, chain := range created {
			new_chain := parser.Chain{Name: chain.Name, Link: chain.Link, Target: chain.Target}
			new_chain.Id = strings.Replace(uuid.New().String(), "-", "", -1)
			new_chain.Steps = append(new_chain.Steps, chain.Steps...)

			if exist {
				conf.Workflows[0].Chains = append(conf.Workflows[0].Chains[:ind], append([]parser.Chain{new_chain}, conf.Workflows[0].Chains[ind:]...)...)
			} else {
				conf.Workflows[0].Chains[ind] = new_chain
			}

			creChains = append(creChains, new_chain)
		}

		chains_names := []string{}
		for _, ch := range conf.Workflows[0].Chains {
			chains_names = append(chains_names, ch.Name)
		}

		runChains := append(updAffected, creChains...)
		if len(delAffected) == 0 && len(runChains) == 0 {
			fmt.Println("\nnothing to do...\n")
		}

		for _, chain := range runChains {
			if chain.Link != "" && exist && Find(oper_chains_names, chain.Link) == -1 {
				link_chain := conf.Workflows[0].Chains[Find(chains_names, chain.Link)]
				
				for _, res := range link_chain.Results {
					if Find(storagecols, res.Id) == -1 {
						runChains = append([]parser.Chain{link_chain}, runChains...)
					}
				}
			}
		}

		// * handle IMPORT *
		columns := []string{}

		import_names := []string{}
		for _, imp := range conf.Import.Columns {
			import_names = append(import_names, imp.Name)
		}

		fetch_cols := []string{}
		for _, chain := range runChains {
			if chain.Link == "" {
				base_chain := parser.Chain{}
				for _, ch := range config.Workflows[0].Chains {
					if ch.Name == chain.Name {
						base_chain = ch
					}
				}

				for _, t := range base_chain.Target {
					ind := Find(import_names, t)

					if ind == -1 || Find(storagecols, conf.Import.Columns[ind].Id) == -1 {
						fetch_cols = append(fetch_cols, t)
					}
				}
			}
		}

		cols := map[string]string{}
		for _, e := range conf.Import.Columns {
			cols[e.Name] = e.Id
		}

		if len(fetch_cols) > 0 {
			out, err := execute.RunData(home, "fetch", execute.DataArgs{Name: config.Name, Columns: fetch_cols})
			if err != nil {
				fmt.Println("Error: ", err)
			}

			ir := parser.ImportResp{}
			ir.ParseImportResp(out)
			for _, col := range ir.Resp {
				ind := Find(import_names, col.Name)
				if ind != -1 {
					conf.Import.Columns[ind] = col
				} else {
					conf.Import.Columns = append(conf.Import.Columns, col)
				}
				cols[col.Name] = col.Id
			}



			fmt.Println("\ndata fetched from dataset:")
			for _, c := range fetch_cols {
				fmt.Printf("  - %s (%s)\n", c, cols[c])
			}
		}

		// * handle CALCULATIONS *

		if len(runChains) > 0 {
			fmt.Println("\ncalculated chains:")
		}
		affectedGroups := differ.GroupChains(runChains)
		for gi, group := range affectedGroups {
			wg := new(sync.WaitGroup)

			for i, chain := range group {
				targets := []string{}

				// find linked chain
				if chain.Link != "" {
					var linked_chain parser.Chain
					
					for _, chainToLink := range conf.Workflows[0].Chains {
						if chainToLink.Name == chain.Link {
							linked_chain = chainToLink
						}
					}
					if len(chain.Target) == 0 {
						for _, r := range linked_chain.Results {
							targets = append(targets, r.Id)
						}
					} else {
						for _, r := range linked_chain.Results {
							for _, t := range chain.Target {
								if t == r.Name {
									targets = append(targets, r.Id)
								}
							}
						}
					}
				} else {
					base_chain := parser.Chain{}
					for _, ch := range config.Workflows[0].Chains {
						if ch.Name == chain.Name {
							base_chain = ch
						}
					}
					for _, t := range base_chain.Target {
						targets = append(targets, cols[t])
					}
				}
				affectedGroups[gi][i].Target = targets
			}

			var mutex sync.Mutex

			for i, chain := range group {
				wg.Add(1)
				go func(chain parser.Chain, i int) {

					out, err := execute.RunChain(home, parser.Chain{Id: chain.Id, Name: chain.Name, Target: chain.Target, Steps: chain.Steps, Link: chain.Link})
					
					if err != nil {
						fmt.Println("Error: ", err)
					}

					cr := parser.ImportResp{}
					cr.ParseImportResp(out)
					chain.Results = cr.Resp
					mutex.Lock()
					for i, origChain := range conf.Workflows[0].Chains {
						if chain.Id == origChain.Id {
							conf.Workflows[0].Chains[i] = chain
							break
						}
					}
					mutex.Unlock()

					runType := "update"
					for _, r := range created {
						if r.Name == chain.Name {
							runType = "create"
							break
						}
					}

					PrintResult(chain, cr, runType)

					for _, r := range chain.Results {
						columns = append(columns, r.Name)
					}

					wg.Done()
				}(chain, i)
			}
			wg.Wait()
		}

		// *handle RESULT STORAGE *

		targets := []string{}
		for _, chain := range conf.Workflows[0].Chains {
			for i := range chain.Target {
				targets = append(targets, chain.Target[i])
			}
		}

		all_affected := []string{}
		for _, ch := range runChains {
			all_affected = append(all_affected, ch.Name)
		}
		results := []string{}
		for _, chain := range conf.Workflows[0].Chains {
			if Find(all_affected, chain.Name) != -1 {
				resloop: for i := range chain.Results {
					for _, t := range targets {
						if t == chain.Results[i].Id {
							break resloop
						}
					}
					results = append(results, chain.Results[i].Id)
				}
			}
		}

		if len(results) > 0 {
		 	out, err := execute.RunData(home, "store", execute.DataArgs{Name: config.Name, Results: results, Key: config.Workflows[0].Name})
			if err != nil {
				fmt.Println("Error: ", err)
			}

			fmt.Printf("\ncalculated columns of '%v' workspace (current run):\n\n", config.Workflows[0].Name)
			for _, line := range strings.Split(string(out), "\n") {
				fmt.Printf("  %s\n", line)
			}
		}

		// * handle CLEANUP *

		err = parser.WriteConfig(config, ".dac/config/.build.yml")
		if err != nil {
			log.Fatal(err)
		}

		err = parser.WriteConfig(conf, ".dac/config/.build.wide.yml")
		if err != nil {
			log.Fatal(err)
		}


		if !*runIntact {
			reply, err := kvs.Info()
			if err != nil {
				fmt.Println("Error: ", err)
			}
			cmd := exec.Command("kill", "-9", reply.Pid)
			err = cmd.Run()
			if err != nil {
				fmt.Println("Error: ", err)
			}
			fmt.Println("project engine closed...")
		}
	
	case "check":

		checkCmd.Parse(os.Args[2:])
		config, err := parser.Parse() // parsed raw config
		if err != nil {
			log.Fatal(err)
		}

		out, err := execute.RunData(home, "check", execute.DataArgs{Name: config.Name, Key: *checkKey})
		if err != nil {
			fmt.Println("Error: ", err)
		}
		fmt.Println()
		for _, line := range strings.Split(string(out), "\n") {
			fmt.Printf("  %s\n", line)
		}


	case "stop":

		var kvs client.Rpc_client
		err := kvs.Connect()
		if err != nil {
			fmt.Println("Error: ", err)
		} else {
			reply, err := kvs.Info()
			if err != nil {
				fmt.Println("Error: ", err)
			}
			cmd := exec.Command("kill", "-9", reply.Pid)
			err = cmd.Run()
			if err != nil {
				fmt.Println("Error: ", err)
			}
			fmt.Println("\nproject engine closed...")
			fmt.Println("\ninteractive mode closed...")
		}
	case "clear":

		var kvs client.Rpc_client
		err := kvs.Connect()
		if err != nil {
			fmt.Println("Error: ", err)
		} else {
			reply, err := kvs.Info()
			if err != nil {
				fmt.Println("Error: ", err)
			}
			fmt.Printf("reply: %+v\n", reply)
			cmd := exec.Command("kill", "-9", reply.Pid)
			err = cmd.Run()
			if err != nil {
				fmt.Println("Error: ", err)
			}
		}

		fmt.Println("clearing: ")
		fmt.Println(" \u2022 .build.yml")
		err = os.Remove("./.dac/config/.build.yml")
		if err != nil {
			fmt.Println("Error: ", err)
		}
		fmt.Println(" \u2022 .build.wide.yml")
		err = os.Remove(".dac/config/.build.wide.yml")
		if err != nil {
			fmt.Println("Error: ", err)
		}

	}
}
