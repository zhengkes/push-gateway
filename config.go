package main

import (
	"bytes"
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
)

var config *confYAML

type confYAML struct {
	Remote   remoteConf `yaml:"remote"`
	Duration int        `yaml:"duration"`
}

//远端
type remoteConf struct {
	Addresses []string `yaml:"addresses"`
	Ident     string   `yaml:"ident"`
}

func isExist(fp string) bool {
	_, err := os.Stat(fp)
	return err == nil || os.IsExist(err)
}
func isFile(fp string) bool {
	f, e := os.Stat(fp)
	if e != nil {
		return false
	}
	return !f.IsDir()
}

func Parse(confs ...string) error {
	if len(confs) == 0 {
		viper.SetConfigType("yaml")
		viper.SetDefault("remote.addresses", []string{"10.160.0.173:5811"})
		viper.SetDefault("remote.ident", "10.160.0.173")
		viper.SetDefault("duration", 20)
		err := viper.Unmarshal(&config)
		return err
	}
	conf := confs[0]
	if !isExist(conf) {
		return fmt.Errorf("%s not exists", conf)
	}
	if !isFile(conf) {
		return fmt.Errorf("%s not file", conf)
	}
	bs, err := ioutil.ReadFile(conf)
	if err != nil {
		return fmt.Errorf("cannot read yml[%s]: %v", conf, err)
	}

	viper.SetConfigType("yaml")
	err = viper.ReadConfig(bytes.NewBuffer(bs))
	if err != nil {
		return fmt.Errorf("cannot read yml[%s]: %v", conf, err)
	}
	err = viper.Unmarshal(&config)
	if err != nil {
		return fmt.Errorf("cannot read yml[%s]: %v", conf, err)
	}
	return err
}
