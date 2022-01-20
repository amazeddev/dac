package main

import (
	"dac/calculate"
	"dac/client"
	"dac/differ"
	"dac/execute"
	"dac/interact"
	"dac/parser"
	"dac/present"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
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
		present.PrintHelp()
	}

	commands := []string{"init", "list", "-h", "help", "import", "validate", "compare", "run", "check", "stop", "clear"}
	if Find(commands, os.Args[1]) == -1 {
		fmt.Printf("dac: '%s' is not a dac command; see 'dac -h' for help\n", os.Args[1])
	}

	initCmd := flag.NewFlagSet("init", flag.ExitOnError)

	valCmd := flag.NewFlagSet("validate", flag.ExitOnError)

	comCmd := flag.NewFlagSet("compare", flag.ExitOnError)
	comWide := comCmd.Bool("v", false, "# verbose, wheather to show full chain data")
	comAll := comCmd.Bool("a", false, "# all, wheather to show all chain, also not changed ones")

	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	runIntact := runCmd.Bool("i", false, "# interactive mode, not clear storage after run")
	runForce := runCmd.Bool("f", false, "# force mode, ignore previous calculations & run all")
	runWflow := runCmd.String("w", "", "# workflow, run only specified workflow")

	checkCmd := flag.NewFlagSet("check", flag.ExitOnError)
	checkKey := checkCmd.String("k", "", "# key, stored data table name")

	home, _ := os.UserHomeDir()

	current, _ := os.Getwd()

	switch os.Args[1] {

	case "init":
		initCmd.Parse(os.Args[2:])
		present.PrintLogo()

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
			configImport.Path = dataSrcPath
		}

		baseConfig := parser.Config{
			Name:    projName,
			Engine:  "python",
			Import:  configImport,
			Workflows: []parser.Workflow{{Name: "base", Chains: []parser.Chain{}}},
		}

		err = parser.WriteConfig(baseConfig, fmt.Sprintf("%v/%v", dirName, "main.yml"))
		if err != nil {
			fmt.Println("\u1F4A5 Error: ", err)
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
		filepath, _ := filepath.Abs(fmt.Sprintf("%s/%s", current, config.Import.Path))
		
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
		fmt.Print("validation: \n\n")

		consStruct := true
		config, err := parser.Parse() // parsed raw config
		if err != nil {
			consStruct = false
			fmt.Println("Error: ", err)
		}
		fmt.Println("base checks:")
		fmt.Printf("  [%s] parse main.yml\n", present.PrintCheck(consStruct))
		fmt.Printf("  [%s] configuration have name\n", present.PrintCheck(config.Name != ""))
		fmt.Printf("  [%s] configuration have import statement\n", present.PrintCheck(config.Import.Path != ""))
		fmt.Printf("  [%s] configuration have workflows\n", present.PrintCheck(len(config.Workflows) > 0))

		for _, w := range config.Workflows {
			fmt.Print("\nworkflow ", w.Name, " checks:\n")
			fmt.Printf("  [%s] workflow have name\n", present.PrintCheck(w.Name != ""))
			fmt.Printf("  [%s] workflow have chains\n", present.PrintCheck(len(w.Chains) > 0))
		}

	
	case "compare":
		comCmd.Parse(os.Args[2:])

		config, err := parser.Parse() // parsed raw config
		if err != nil {
			log.Fatal(err)
		}
		exist_config := []parser.Chain{}

		for widx := range config.Workflows {
			fmt.Println("workflow: ", config.Workflows[widx].Name)
			if _, err := os.Stat(".dac/config/.build.yml"); err == nil {
				parsed, _ := parser.GetConfig(".dac/config/.build.yml")
				exist_config = parsed.Workflows[widx].Chains
			}

			deleted, updated, created := differ.FindDiffs(exist_config, config.Workflows[widx].Chains)

			validateChains := make([]ChainValiObj, len(config.Workflows[widx].Chains))
			for i, e := range config.Workflows[widx].Chains {
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
				if !*comAll && e.T == "exist" {
					continue
				}
				present.PrintChain(e.Chain, e.T, *comWide)
			}

			fmt.Printf("\n  created: %v, updated: %v, deleted: %v (of %v chains total)\n\n", len(created), len(updated), len(deleted), len(config.Workflows[widx].Chains))

		}

	case "run":

		runCmd.Parse(os.Args[2:])
		config, err := parser.Parse() // parsed raw config
		if err != nil {
			log.Fatal(err)
		}

		var kvs client.Rpc_client
		err = execute.StartEngin(&kvs, config.Name, home)
		if err != nil {
			log.Fatal(err)
		}

		storagecols, err := kvs.List(struct{}{})		
		if err != nil {
			log.Fatal(err)
		}
		
		rerun := false
		var conf parser.Config // empty object for wide config
		exist_config := parser.Config{}
		exist_config.Name = config.Name
		exist_config.Engine = config.Engine
		exist_config.Import.Path = config.Import.Path

		conf.Name = config.Name
		conf.Engine = config.Engine
		conf.Import.Path = config.Import.Path
		conf.Workflows = []parser.Workflow{}

		if _, err := os.Stat(".dac/config/.build.yml"); err == nil && !*runForce {
			rerun = true
			exist_config, _ = parser.GetConfig(".dac/config/.build.yml")
		} 

		if _, err := os.Stat(".dac/config/.build.wide.yml"); err == nil && !*runForce {
			conf, _ = parser.GetConfig(".dac/config/.build.wide.yml")

		}

		
		fmt.Println("\nrun modes: ")
		fmt.Println(" \u2022 interactive: ", present.PrintCheck(*runIntact))
		fmt.Println(" \u2022 force:       ", present.PrintCheck(*runForce))
		// fmt.Println(" \u2022 workflow:    ", PrintCheck(*runWflow != ""))
		fmt.Println()

		workflowloop: for widx := range config.Workflows {
			if *runWflow != "" && *runWflow != config.Workflows[widx].Name {
				continue
			}

			conf_wfs := []string{}
			for _, w := range conf.Workflows {
				conf_wfs = append(conf_wfs, w.Name)
			}
			cwf_idx := Find(conf_wfs, config.Workflows[widx].Name)

			wide_workflow := parser.Workflow{}

			if rerun && cwf_idx != -1 {
				wide_workflow.Name = conf.Workflows[cwf_idx].Name
				wide_workflow.Data = conf.Workflows[cwf_idx].Data
				wide_workflow.Chains = append([]parser.Chain{}, conf.Workflows[cwf_idx].Chains...)
			} else {
				wide_workflow.Name = config.Workflows[widx].Name
				wide_workflow.Data = config.Workflows[widx].Data
				wide_workflow.Chains = append(wide_workflow.Chains, config.Workflows[widx].Chains...)
			}

			fmt.Println("workflow: ", string("\033[36m"), wide_workflow.Name, string("\033[0m"))

			var deleted map[int]parser.Chain
			var updated map[int]parser.Chain
			var created map[int]parser.Chain

			exist_chains := []parser.Chain{}
			exist_wfs := []string{}
			for _, w := range exist_config.Workflows {
				exist_wfs = append(exist_wfs, w.Name)
			}
			if rerun {
				workflow_idx := Find(exist_wfs, wide_workflow.Name)
				if workflow_idx != -1 {
					exist_chains = exist_config.Workflows[workflow_idx].Chains
				}
			}

			deleted, updated, created = differ.FindDiffs(exist_chains, config.Workflows[widx].Chains)

			oper_chains_names := []string{}
			for _, chain := range created {
				oper_chains_names = append(oper_chains_names, chain.Name) 
			}
			for _, chain := range updated {
				oper_chains_names = append(oper_chains_names, chain.Name) 
			}

			// * handle DELETED *
			delChains := []parser.Chain{}

			cols_to_delete := []string{}
			storage_to_delete := []string{}

			if len(deleted) > 0 {
				filtered_chains := wide_workflow.Chains[:0]
				mainloop: for idx, chain := range wide_workflow.Chains {
					for _, dch := range deleted {
						if dch.Name == chain.Name {
							delChains = append(delChains, wide_workflow.Chains[idx])
							continue mainloop
						}
					}
					filtered_chains = append(filtered_chains, chain)
				}
		
				wide_workflow.Chains = filtered_chains
			}

			for _, ch := range delChains {
				for _, e := range ch.Results {
					storage_to_delete = append(storage_to_delete, e.Id)

					if Find(cols_to_delete, e.Name) == -1 {
						cols_to_delete = append(cols_to_delete, e.Name)
					}
				}
			}

			// * handle FORCE CLEAR *

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

			updAffected := differ.ConnectedChainsMulti(updChains, config.Workflows[widx].Chains)

			for i, ch := range updAffected {
				var ind int
			confloop:
				for i, c := range wide_workflow.Chains {
					if c.Name == ch.Name {
						ind = i
						for _, e := range c.Results {
							storage_to_delete = append(storage_to_delete, e.Id)
						}
						break confloop
					}
				}

				new_chain := parser.Chain{Name: ch.Name, Link: ch.Link, Target: ch.Target, Group: ch.Group}
				if wide_workflow.Chains[ind].Id == "" {
					new_chain.Id = strings.Replace(uuid.New().String(), "-", "", -1)
				} else {
					new_chain.Id = wide_workflow.Chains[ind].Id
				}
				new_chain.Steps = append(new_chain.Steps, ch.Steps...)


				wide_workflow.Chains[ind] = new_chain
				updAffected[i] = new_chain
			}

			// * handle CREATED *
			creChains := []parser.Chain{}

			for ind, chain := range created {
				new_chain := parser.Chain{Name: chain.Name, Link: chain.Link, Target: chain.Target, Group: chain.Group}
				new_chain.Id = strings.Replace(uuid.New().String(), "-", "", -1)
				new_chain.Steps = append(new_chain.Steps, chain.Steps...)

				if rerun && cwf_idx != -1 {
					wide_workflow.Chains = append(wide_workflow.Chains[:ind], append([]parser.Chain{new_chain}, wide_workflow.Chains[ind:]...)...)
				} else {
					wide_workflow.Chains[ind] = new_chain
				}

				creChains = append(creChains, new_chain)
			}

			chains_names := []string{}
			for _, ch := range wide_workflow.Chains {
				chains_names = append(chains_names, ch.Name)
			}

			runChains := append(updAffected, creChains...)
			if len(delChains) + len(runChains) == 0 {
				fmt.Print("\n  \u2755 no changes in configuration for workflow, no action to do...\n")
				fmt.Print("     to run all chains anyway, try to use force mode 'dac run -f [-w=workflow] [-i]'\n\n")

				continue workflowloop
			}

			if rerun {
				for _, chain := range runChains {
					if chain.Link != "" && Find(oper_chains_names, chain.Link) == -1 {
						runChains = append(differ.FindLink(chain, wide_workflow, chains_names, oper_chains_names, storagecols), runChains...)
					}
				}
			}
			// create chain map (map[chain.Name]chain."Info")
			chain_map := map[string]parser.ChainMapElem{}
			linked_chans := []string{}
			for cind, chain := range wide_workflow.Chains {
				chain_map[chain.Name] = parser.ChainMapElem{Idx: cind, Link: chain.Link}
				if chain.Link != "" && Find(linked_chans, chain.Link) == -1 {
					linked_chans = append(linked_chans, chain.Link)
				}
			}

			// * handle IMPORT *

			import_names := []string{}
			for _, imp := range conf.Import.Columns {
				import_names = append(import_names, imp.Name)
			}

			fetch_cols := []string{}
			for _, chain := range runChains {
				if chain.Link == "" {
					base_chain := wide_workflow.Chains[chain_map[chain.Name].Idx]

					for _, t := range append(base_chain.Target, base_chain.Group...) {
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

			fmt.Println("\n  workflow checks: ")
			out, err := execute.RunData(home, "fetchInfo", execute.DataArgs{Name: config.Name, Key: wide_workflow.Data})
			if err != nil {
				fmt.Println("Error: ", err)
			}

			fetchInfoResp := parser.ImportResp{}
			fetchInfoResp.ParseImportResp(out)

			fetch_info_cols := []string{}
			fetch_ok := true
			for _, col := range fetchInfoResp.Resp {
				fetch_info_cols = append(fetch_info_cols, col.Name)
			}
			for _, col := range fetch_cols {
				if Find(fetch_info_cols, col) == -1 {
					fetch_ok = false
					break
				}
			}
			fmt.Printf("    [%s] necesary base columns available\n", present.PrintCheck(fetch_ok))

			link_ok := true
			for _, lc := range linked_chans {
				if Find(chains_names, lc) == -1 {
					link_ok = false
					break
				}
			}
			fmt.Printf("    [%s] all linked chains available\n", present.PrintCheck(link_ok))

			inputFuncs := []string{"ffill", "bfill", "cfill", "meanfill"}
			encodeFuncs := []string{"encode", "digitize"}
			transFuncs := []string{"normalize"}

			funcs_ok := true
			for _, chain := range wide_workflow.Chains {
				for _, step := range chain.Steps {
					if step.StepType == "" {
						f := strings.Split(step.Function, ".")
						switch f[0]{
						case "input":
							if Find(inputFuncs, f[1]) == -1 {
								funcs_ok = false
								break
							}
						case "encode":
							if Find(encodeFuncs, f[1]) == -1 {
								funcs_ok = false
								break
							}
						case "trans":
							if Find(transFuncs, f[1]) == -1 {
								funcs_ok = false
								break
							}
						}
					}
				}
			}
			fmt.Printf("    [%s] correct steps functions\n", present.PrintCheck(funcs_ok))

			if !fetch_ok || !link_ok || !funcs_ok {
				fmt.Println("\n  \u2757 one or more checks failed, workflow processing could not continue...")
				continue workflowloop
			}

			if len(fetch_cols) > 0 {
				out, err := execute.RunData(home, "fetch", execute.DataArgs{Name: config.Name, Columns: fetch_cols, Key: wide_workflow.Data})
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

				fetch_errors := []string{}
				for _, e := range ir.Errors {
					fetch_errors = append(fetch_errors, e.Name)
				}

				fmt.Println("\n  data fetched from dataset:")
				for _, c := range fetch_cols {
					if Find(fetch_errors, c) == -1 {
						fmt.Printf("    \u2022 %s (%s)\n", c, cols[c])
					}
				}

				if len(fetch_errors) > 0 {
					fmt.Println("  fetch errors:")
					for _, e := range ir.Errors {
						fmt.Printf("    \u2757 %s \n", e.Error)
					}
				}
			}

			import_map := map[string]parser.ImportMapElem{}
			for i, col := range conf.Import.Columns {
				import_map[col.Name] = parser.ImportMapElem{Idx: i, Id: col.Id}
			}

			// * handle CALCULATIONS *

			no_run_chains := []string{}
			proceed_on_fail := false

			if len(runChains) + len(delChains) > 0 {
				fmt.Print("\n  updated chains:\n")
			}
			affectedGroups := differ.GroupChains(runChains)
			for _, group := range affectedGroups {

				if len(no_run_chains) > 0 && !proceed_on_fail {
					fmt.Println("\n  \u2755 one or more chain failed, result data will be missing some columns")
					proceed_on_fail = interact.ConfirmPrompt("  would you like to proceed and try to complete rest?")
					if !proceed_on_fail {
						fmt.Println("\nexecution stoped by user...")
								
						if !*runIntact {
							execute.StopEngine(kvs)
							if err != nil {
								fmt.Println("Error: ", err)
							}
						}
						os.Exit(0)
					}
				}

				calcchan := make(chan parser.CalcResults, 1)
				wg := new(sync.WaitGroup)

				calculated := map[string]parser.CalcResults{}

				var mutex sync.Mutex

				for i, chain := range group {
					
					if len(chain.Group) == 0 {
						wg.Add(1)
						go calculate.CalculateChain(
							chain, i, "", wide_workflow, chain_map, import_map, no_run_chains, home, wg, calcchan,
						)
					} else {
						wg.Add(len(chain.Group))
						for i, ge := range chain.Group {
							go calculate.CalculateChain(
								chain, i, ge, wide_workflow, chain_map, import_map, no_run_chains, home, wg, calcchan,
							)
						}
					}

					go func() {
						for cr := range calcchan {
							runType := "update"
							for _, r := range created {
								if r.Name == cr.Chain.Name {
									runType = "create"
									break
								}
							}
							cr.RunType = runType
							mutex.Lock()
							if _, ok := calculated[cr.Chain.Name]; ok {
								tmp_calc := calculated[cr.Chain.Name]
								tmp_calc.Responses = append(tmp_calc.Responses, cr.Responses...)
								tmp_calc.Errors = append(tmp_calc.Errors, cr.Errors...)
								calculated[cr.Chain.Name] = tmp_calc
								wide_workflow.Chains[chain_map[cr.Chain.Name].Idx].Results = append(wide_workflow.Chains[chain_map[cr.Chain.Name].Idx].Results, cr.Responses...)
							} else {
								calculated[cr.Chain.Name] = cr
								wide_workflow.Chains[chain_map[cr.Chain.Name].Idx].Results = cr.Responses
								wide_workflow.Chains[chain_map[cr.Chain.Name].Idx].Results = cr.Responses
							}
							if !cr.Success {
								no_run_chains = append(no_run_chains, cr.Chain.Name)
							}
							mutex.Unlock()
							wg.Done()
						}
					}()		
				}
				wg.Wait()
				for _, calc := range calculated {
					present.PrintResult(calc.Chain, calc.Responses, calc.Errors, calc.RunType, calc.Status)
				}
			}

			// * handle CLEANUP *

			wf_idx := Find(exist_wfs, wide_workflow.Name)

			exist_chains_names := []string{}
			for _, ch := range exist_chains {
				exist_chains_names = append(exist_chains_names, ch.Name)
			}
			
			filtered_wf := parser.Workflow{}
			filtered_wf.Name = config.Workflows[widx].Name
			filtered_wf.Data = config.Workflows[widx].Data

			for _, ch := range config.Workflows[widx].Chains {
				if Find(no_run_chains, ch.Name) == -1 {
					filtered_wf.Chains = append(filtered_wf.Chains, ch)
				} else {
					exist_idx := Find(exist_chains_names, ch.Name)
					if exist_idx != -1 {
						filtered_wf.Chains = append(filtered_wf.Chains, exist_chains[exist_idx])
					}
				}
			}
			
			if wf_idx == -1 {
				exist_config.Workflows = append(exist_config.Workflows, filtered_wf)
			} else {
				exist_config.Workflows[wf_idx] = filtered_wf
			}

			err = parser.WriteConfig(exist_config, ".dac/config/.build.yml")
			if err != nil {
				log.Fatal(err)
			}

			conf_wfs_names := []string{}
			for _, w := range conf.Workflows {
				conf_wfs_names = append(conf_wfs_names, w.Name)
			}
			
			wf_idx = Find(conf_wfs_names, wide_workflow.Name)
			
			if wf_idx == -1 {
				conf.Workflows = append(conf.Workflows, wide_workflow)
			} else {
				conf.Workflows[wf_idx] = wide_workflow
			}

			err = parser.WriteConfig(conf, ".dac/config/.build.wide.yml")
			if err != nil {
				log.Fatal(err)
			}

			for _, col_id := range storage_to_delete {
					args := client.DelArgs{Key: col_id}
					_, err := kvs.Delete(args)
					if err != nil {
						panic(err)
					}
			}

			
			if len(cols_to_delete) > 0 || *runForce {
				_, err = execute.RunData(home, "remove", execute.DataArgs{Name: config.Name, Columns: cols_to_delete, Key: wide_workflow.Name})
				if err != nil {
					fmt.Println("Error: ", err)
				}
			}

			for _, ch := range delChains {
				present.PrintResult(ch, ch.Results, []parser.ErrorElem{}, "delete", "completed")
			}

			// * handle RESULT STORAGE *
			
			all_affected := []string{}
			for _, ch := range runChains {
				all_affected = append(all_affected, ch.Name)
			}

			results := []string{}
			for _, chain := range wide_workflow.Chains {
				if Find(linked_chans, chain.Name) == -1 &&  Find(all_affected, chain.Name) != -1 {
					for _, r := range chain.Results {
						results = append(results, r.Id)
					}
				}
			}

			if len(results) > 0 {
				out, err := execute.RunData(home, "store", execute.DataArgs{Name: config.Name, Results: results, Key: wide_workflow.Name})
				if err != nil {
					fmt.Println("Error: ", err)
				}

				fmt.Printf("\n  calculated columns of '%v' workspace (current run):\n\n", wide_workflow.Name)
				for _, line := range strings.Split(string(out), "\n") {
					fmt.Printf("   %s\n", line)
				}
			}

		}


		if !*runIntact {
			execute.StopEngine(kvs)
			if err != nil {
				fmt.Println("Error: ", err)
			}
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
	
	case "help":
		present.PrintHelp()
	}
}
