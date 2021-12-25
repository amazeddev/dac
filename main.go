package main

import (
	"bytes"
	"dac/client"
	"dac/differ"
	"dac/parser"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jinzhu/copier"
	"gopkg.in/yaml.v2"
)

type GetArgs struct {
	Key  string
	Base bool
}

type ColObj struct {
	Name string
	Type string
	Data []interface{}
	Pres []int
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

func PrintChain(chain parser.Chain, t string, wide bool) {
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
	fmt.Println(color)
	ymlBytes, _ := yaml.Marshal(chain)
	ymlSlice := strings.Split(string(ymlBytes), "\n")
	linePerChain := 1
	if wide {
		linePerChain = len(ymlSlice) - 1
	}
	for i := range ymlSlice[:linePerChain] {
		fmt.Printf(" %v %v %v\n", color, ymlSlice[i], colorReset)
	}
}

func main() {

	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	initDir := initCmd.String("dir", ".", "# specify project dir")

	startCmd := flag.NewFlagSet("start", flag.ExitOnError)

	valCmd := flag.NewFlagSet("validate", flag.ExitOnError)
	valWide := valCmd.Bool("wide", false, "# wheather to show full chain data")
	valAll := valCmd.Bool("all", false, "# wheather to show not changed chains")

	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	// runChain := runCmd.String("chain", "", "# run single chain calculations")
	runForce := runCmd.Bool("force", false, "# rerun all chains even if nothing changed")

	home, _ := os.UserHomeDir()
	// fmt.Println(home)
	// folderInfo, err := os.Stat(fmt.Sprintf("%v/%v", home, ".dac"))
	// if os.IsNotExist(err) {
	// 		log.Fatal("Folder ~/.dac does not exist.")
	// }
	// log.Printf("%+v", folderInfo)

	switch os.Args[1] {
	case "init":
		initCmd.Parse(os.Args[2:])
		dirName := *initDir

		_ = os.MkdirAll(fmt.Sprintf("%v/%v", dirName, ".dac/config"), 0755)
		_ = os.MkdirAll(fmt.Sprintf("%v/%v", dirName, ".dac/data"), 0755)
		_ = os.Mkdir(fmt.Sprintf("%v/%v", dirName, "functions"), 0755)
		_ = os.MkdirAll(fmt.Sprintf("%v/%v", dirName, "modules"), 0755)

		path, _ := os.Getwd()
		v := strings.Split(path, "/")
		projName := dirName

		if dirName == "." {
			projName = v[len(v)-1]
		}

		baseConfig := parser.Config{
			Name:    projName,
			Engine:  "python",
			Import:  parser.Import{},
			Workflows: []parser.Workflow{{Name: "base", Chains: []parser.Chain{}}},
		}

		err := parser.WriteConfig(baseConfig, fmt.Sprintf("%v/%v", dirName, "main.yml"))
		if err != nil {
			log.Fatal(err)
		}

	case "start":

		startCmd.Parse(os.Args[2:])
		config, err := parser.Parse()
		if err != nil {
			log.Fatal(err)
		}

		cmd := exec.Command(fmt.Sprintf("%v/%v", home, ".dac/kvstore"))
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

		var kvs client.Rpc_client
		err = kvs.Connect()
		if err != nil {
			log.Fatal(err)
		}

		reply, err := kvs.SetPid(client.PidArgs{Name: config.Name, Pid: fmt.Sprint(cmd.Process.Pid)})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("pid stored: %v\n", reply)

		var conf parser.Config // empty object for wide config
		conf.Name = config.Name
		conf.Engine = config.Engine
		conf.Import = config.Import

		targets := []string{}
		for _, ch := range config.Workflows[0].Chains {
			if ch.Link == "" {
				targets = append(targets, ch.Target...)
			}
		}
		json_targets, _ := json.Marshal(targets)

		cmd = exec.Command("python3", fmt.Sprintf("%v/%v", home, ".dac/lib/fetch.py"), string(json_targets), config.Name)

		out, _ := cmd.Output()

		fmt.Println(string(out))
		ir := parser.ImportResp{}
		ir.ParseImportResp(out)
		conf.Import.Columns = ir.Resp

		err = parser.WriteConfig(conf, ".dac/config/.build.wide.yml")
		if err != nil {
			log.Fatal(err)
		}

	case "list":
		var kvs client.Rpc_client
		err := kvs.Connect()
		if err != nil {
			fmt.Println("!")
			log.Fatal(err)
		}

		reply, err := kvs.List(struct{}{})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%+v\n", reply)

	case "import":

		config, err := parser.Parse() // parsed raw config
		if err != nil {
			log.Fatal(err)
		}

		cmd := exec.Command("python3", fmt.Sprintf("%v/%v", home, ".dac/lib/import.py"), config.Import.Dataset, config.Name)
		var stdBuffer bytes.Buffer
		mw := io.MultiWriter(os.Stdout, &stdBuffer)

		cmd.Stdout = mw
		cmd.Stderr = mw

		// Execute the command
		if err := cmd.Run(); err != nil {
			log.Panic(err)
		}

		log.Println(stdBuffer.String())

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

		fmt.Printf("\ncreated: %v, updated: %v, deleted: %v (of %v chains total)", len(created), len(updated), len(deleted), len(config.Workflows[0].Chains))

	case "run":

		start := time.Now()

		var kvs client.Rpc_client
		err := kvs.Connect()
		if err != nil {
			log.Fatal(err)
		}

		runCmd.Parse(os.Args[2:])
		config, err := parser.Parse() // parsed raw config
		if err != nil {
			log.Fatal(err)
		}

		var conf parser.Config // empty object for wide config

		var deleted map[int]parser.Chain
		var updated map[int]parser.Chain
		var created map[int]parser.Chain

		conf, _ = parser.GetConfig(".dac/config/.build.wide.yml")
		exist_chains := []parser.Chain{}

		if _, err := os.Stat(".dac/config/.build.yml"); err == nil && !*runForce {
			fmt.Println("config/.build.yml exist")

			old_config, _ := parser.GetConfig(".dac/config/.build.yml")
			exist_chains = old_config.Workflows[0].Chains
		} else if errors.Is(err, os.ErrNotExist) || *runForce {
			fmt.Println("config/.build.yml does *not* exist")
			copier.Copy(&conf.Workflows, &config.Workflows)
		} else {
			log.Fatal(err)
		}

		deleted, updated, created = differ.FindDiffs(exist_chains, config.Workflows[0].Chains)

		// if *runChain != "" {
		// 	// * if calculate single chain

		// 	var chainName string
		// 	fmt.Println("calculate chain ", *runChain)
		// 	// check if chain to run exist
		// 	for i := range config.Workflows[0].Chains {
		// 		if config.Workflows[0].Chains[i].Name == *runChain {
		// 			chainName = *runChain
		// 			break
		// 		}
		// 	}

		// 	fmt.Println(chainName, "@@@")

		// 	// check if chain to run in updated
		// 	newUpdate := map[int]parser.Chain{}

		// 	for i, ch := range updated {
		// 		if ch.Name == chainName {
		// 			newUpdate[i] = ch
		// 		}
		// 	}

		// 	// check if chain to run in created
		// 	newCreated := map[int]parser.Chain{}

		// 	for i, ch := range created {
		// 		if ch.Name == chainName {
		// 			newCreated[i] = ch
		// 		}
		// 	}
		// 	deleted = map[int]parser.Chain{}
		// 	updated = newUpdate
		// 	created = newCreated
		// 	fmt.Println(deleted)
		// 	fmt.Println(updated)
		// 	fmt.Println(created)
		// }

		// * handle DELETED *
		delChains := []parser.Chain{}

		for ind := range deleted {
			delChains = append(delChains, conf.Workflows[0].Chains[ind])
		}

		delAffected := differ.ConnectedChainsMulti(delChains, conf.Workflows[0].Chains)

		for _, ch := range delAffected {
			for _, e := range ch.Results {
				args := client.DelArgs{Key: e.Id, Base: false}
				_, err := kvs.Delete(args)
				if err != nil {
					panic(err)
				}
			}
		}

		for ind := range deleted {
		dinner:
			for i, c := range conf.Workflows[0].Chains {
				if c.Name == deleted[ind].Name {
					conf.Workflows[0].Chains = append(conf.Workflows[0].Chains[:i], conf.Workflows[0].Chains[i+1:]...)
					break dinner
				}
			}
		}

		// * handle UPDATED *
		updChains := []parser.Chain{}

		if len(updated) == 0 && len(created) == 0 {
			fmt.Println("nothing to calculate!")
			return
		}

		for ind := range updated {
			updChains = append(updChains, updated[ind])
		}

		cols := map[string]string{}
		for _, e := range conf.Import.Columns {
			cols[e.Name] = e.Id
		}

		updAffected := differ.ConnectedChainsMulti(updChains, conf.Workflows[0].Chains)

		for i, ch := range updAffected {
			for _, e := range ch.Results {
				args := client.DelArgs{Key: e.Id, Base: false}
				_, err := kvs.Delete(args)
				if err != nil {
					panic(err)
				}
			}
			var ind int
		confloop:
			for i, c := range conf.Workflows[0].Chains {
				if c.Name == ch.Name {
					ind = i
					break confloop
				}
			}
			ch.Id = strings.Replace(uuid.New().String(), "-", "", -1)
			ch.Target = config.Workflows[0].Chains[ind].Target
			if ind > len(conf.Workflows[0].Chains) {
				conf.Workflows[0].Chains = append(conf.Workflows[0].Chains, ch)
			} else {
				conf.Workflows[0].Chains[ind] = ch
			}
			updAffected[i] = ch
		}

		// * handle CREATED *
		creChains := []parser.Chain{}

		for ind, chain := range created {
			chain.Id = strings.Replace(uuid.New().String(), "-", "", -1)

			if ind > len(conf.Workflows[0].Chains) {
				conf.Workflows[0].Chains = append(conf.Workflows[0].Chains, chain)
			} else {
				conf.Workflows[0].Chains = append(conf.Workflows[0].Chains[:ind], append([]parser.Chain{chain}, conf.Workflows[0].Chains[ind:]...)...)
			}
			creChains = append(creChains, chain)
		}

		affectedGroups := differ.GroupChains(append(updAffected, creChains...))
		for gi, group := range affectedGroups {
			wg := new(sync.WaitGroup)

			for i, chain := range group {
				targets := []string{}

				// find linked chain
				if chain.Link != "" {
					var linked_chain parser.Chain
					MapName(conf.Workflows[0].Chains)
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
					for _, t := range chain.Target {
						targets = append(targets, cols[t])
					}
				}
				affectedGroups[gi][i].Target = targets
			}

			var mutex sync.Mutex

			for i, chain := range group {
				wg.Add(1)
				go func(chain parser.Chain, i int) {
					data, _ := json.Marshal(parser.Chain{Id: chain.Id, Name: chain.Name, Target: chain.Target, Steps: chain.Steps, Link: chain.Link})

					cmd := exec.Command("python3", fmt.Sprintf("%v/%v", home, ".dac/lib/chain.py"), string(data))

					out, err := cmd.Output()
					if err != nil {
						fmt.Println(err)
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
					wg.Done()
				}(chain, i)
			}
			wg.Wait()
		}

		err = parser.WriteConfig(config, ".dac/config/.build.yml")
		if err != nil {
			log.Fatal(err)
		}

		err = parser.WriteConfig(conf, ".dac/config/.build.wide.yml")
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("\n!!!!", time.Since(start))

		results := []string{}
		for _, chain := range conf.Workflows[0].Chains {
			for i := range chain.Results {
				results = append(results, chain.Results[i].Id)
			}
		}
		data, _ := json.Marshal(results)

		cmd := exec.Command("python3", fmt.Sprintf("%v/%v", home, ".dac/lib/store.py"), config.Workflows[0].Name, string(data), config.Name)
		var stdBuffer bytes.Buffer
		mw := io.MultiWriter(os.Stdout, &stdBuffer)

		cmd.Stdout = mw
		cmd.Stderr = mw

		// Execute the command
		if err := cmd.Run(); err != nil {
			log.Panic(err)
		}

		log.Println(stdBuffer.String())
	
	case "check":

		config, err := parser.Parse() // parsed raw config
		if err != nil {
			log.Fatal(err)
		}

		cmd := exec.Command("python3", fmt.Sprintf("%v/%v", home, ".dac/lib/check.py"), config.Name)

		cmd.Stdout = os.Stdout

		// Execute the command
		if err := cmd.Run(); err != nil {
			log.Panic(err)
		}


	case "stop":

		var kvs client.Rpc_client
		err := kvs.Connect()
		if err != nil {
			log.Fatal(err)
		}

		reply, err := kvs.Info()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("reply: %v\n", reply)
		cmd := exec.Command("kill", "-9", reply.Pid)
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}

		err = os.Remove("./.dac/config/.build.yml")
		if err != nil {
			log.Fatal(err)
		}
		err = os.Remove(".dac/config/.build.wide.yml")
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("removed")
	}

	fmt.Println("\n\n FINISHED!")
}
