package model

import(
	"fmt"
	"github.com/spf13/viper"
	"os"
)

type Config struct {
	Environment string
	ServiceConfig
	DatabaseConfig
}

type ServiceConfig struct {
	Name string
	Version string
}

type DatabaseConfig struct {
	Host string
	Port int
}

func DefaultConfiguration() *Config {

	viper.SetConfigName("application")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$PWD/config/")
	viperError := viper.ReadInConfig()
	if viperError != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", viperError))
	}

	return &Config{
		Environment: viper.GetString("environment"),
		ServiceConfig: ServiceConfig{
			Name: viper.GetString("serviceName"),
			Version: os.ExpandEnv(viper.GetString("version")),
		},
		DatabaseConfig: DatabaseConfig{
			Host: viper.GetString("mongodbHost"),
			Port: viper.GetInt("mongodbPort"),
		},
	}
}


	

