package shiba

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

var (
	rawFileCfg yaml.Node
	fileCfg    = make(map[string]yaml.Node)
)

func loadConfig(fileName string) error {
	if fileName == "" {
		return nil
	}

	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, &rawFileCfg); err != nil {
		return err
	}

	err = rawFileCfg.Decode(&fileCfg)
	if err != nil {
		return err
	}

	return nil
}
