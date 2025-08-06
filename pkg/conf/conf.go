/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: tooling to load and parse configuration files
 *
**/

package conf

import (
	"fmt"
	"os"
	"strings"

	"github.com/maplerime/cl-query/utils/logging"
	"github.com/maplerime/cl-query/utils/sys"
	"github.com/spf13/viper"
)

// DefaultConfPath default cl-query configuration path
const DefaultConfPath = "/etc/petacloud/cl-query"

var logger = logging.MustGetLogger("utils/conf")

// Configuration ...
type Configuration struct {
	vp *viper.Viper
}

// New Configuration
func New(cName string) (*Configuration, error) {
	logger.Debugf("loading configuration %s ...", cName)
	config := &Configuration{
		vp: viper.New(),
	}
	err := config.init(cName)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func (c *Configuration) init(cName string) error {
	logger.Debugf("Initializing viper for %s ...", cName)
	c.vp.SetConfigName(cName)
	prefix := strings.ToUpper(cName)
	var altPath = os.Getenv(fmt.Sprintf("%s_CFG_PATH", prefix))
	if altPath != "" {
		logger.Debugf("adding alternative configuration path [%s] for %s", altPath, cName)
		c.vp.AddConfigPath(altPath)
	} else {
		c.vp.AddConfigPath("./etc")
	}
	if sys.DirExists(DefaultConfPath) {
		logger.Debugf("adding default configuration path [%s] for %s", DefaultConfPath, cName)
		c.vp.AddConfigPath(DefaultConfPath)
	}

	// handling environment variables
	c.vp.SetEnvPrefix(prefix)
	c.vp.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	c.vp.SetEnvKeyReplacer(replacer)

	err := c.vp.ReadInConfig()
	if err != nil {
		logger.Errorf("Failed to load configuration for %s, error: %+v.", err)
		return err
	} else {
		logger.Infof("Configuration file loaded: %s", c.vp.ConfigFileUsed())
	}
	return nil
}

// Unmarshal configuration
func (c *Configuration) Unmarshal(rawVal interface{}) error { return c.vp.Unmarshal(rawVal) }
