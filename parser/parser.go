package parser

import (
	"fmt"
	"io/ioutil"

	yaml "gopkg.in/yaml.v3"
)

type Step struct {
	Function string						`yaml:"function" json:"function"`
	StepType string						`yaml:"type,omitempty" json:"type,omitempty" default:"standard"`
	Target interface{}				`yaml:"target,omitempty" json:"target,omitempty"`
	Result interface{}				`yaml:"result,omitempty" json:"result,omitempty"`
	Args map[string]interface{} `yaml:"args,omitempty" json:"args,omitempty"`
}

type Chain struct {
	Id string									`yaml:"id,omitempty" json:"id,omitempty"`
	Name string								`yaml:"name,omitempty" json:"name,omitempty"`
	Target []string 					`yaml:"target,omitempty" json:"target,omitempty"`
	Group []string 						`yaml:"group,omitempty" json:"group,omitempty"`
	Steps []Step							`yaml:"steps,omitempty" json:"steps,omitempty"`
	Block string							`yaml:"block,omitempty" json:"block,omitempty"`
	Type string								`yaml:"type,omitempty" json:"type,omitempty"`
	Results []ResultElem 			`yaml:"results,omitempty" json:"results,omitempty"`
	Link string								`yaml:"link,omitempty" json:"link,omitempty"`	
}

type Import struct {
	Type string								`yaml:"type" json:"type"`
	Dataset string						`yaml:"dataset" json:"dataset"`
	Columns []ResultElem			`yaml:"columns,omitempty" json:"columns,omitempty"`
}	

type Workflow struct {
	Name string							`yaml:"name" json:"name"`
	Chains []Chain					`yaml:"chains" json:"chains"`
}	

type Config struct {
	Name string								`yaml:"name" json:"name"`
	Engine string							`yaml:"engine" json:"engine"`
	Import Import							`yaml:"import" json:"import"`
	Workflows []Workflow			`yaml:"workflows" json:"workflows"`
}

func (c *Config) ParseConfig(data []byte) error {
    return yaml.Unmarshal(data, c)
}

func (c *Chain) ParseChain(data []byte) error {
    return yaml.Unmarshal(data, c)
}

type ResultElem struct {
		Id string								`yaml:"id" json:"id"`
		Name string							`yaml:"name" json:"name"`
	}	

type ImportResp struct {
	Resp []ResultElem					`yaml:"resp" json:"resp"`
}

func (i *ImportResp) ParseImportResp(data []byte) error {
  return yaml.Unmarshal(data, i)
}

func Parse() (Config, error) {
	source, err := ioutil.ReadFile("main.yml")
	if err != nil {
		return Config{}, err
	}
	var config Config
	if err := config.ParseConfig(source); err != nil {
		return Config{}, err
	}
	chains := config.Workflows[0].Chains

	for i, chain := range chains {
		if len(chain.Block) != 0  {
			source, _ := ioutil.ReadFile(fmt.Sprintf("%v", chain.Block))
			var newChain Chain
			if err := newChain.ParseChain(source); err != nil {
				return Config{}, err
			}
			config.Workflows[0].Chains[i] = newChain
		}
		// chains[i].Id = strings.Replace(uuid.New().String(), "-", "", -1)
	}

	return config, nil
}


func WriteConfig(config Config, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err	
	}
	
	err = ioutil.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}

	return nil

}

func GetConfig(path string) (Config, error) {
	source, err := ioutil.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var config Config
	if err := config.ParseConfig(source); err != nil {
		return Config{}, err
	}

	return config, nil
}