/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019-2020 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: common configuration structs
 *
**/
package conf

import "time"

// ServerCfg contains config which should be common among all rpc service types.
type ServerCfg struct {
	ListenAddress  string
	ServiceAddress string
	ListenPort     uint16
	ServicePort    uint16
	TLS            TLS
	Profile        Profile
	Log            Log
	Enabled        bool
	AllowedOrigins []string
}

// RPCSvcCfg contains the configuration a common server.
type RPCSvcCfg struct {
	SVCID      string
	HTTPSvcCfg ServerCfg
	GRPCSvcCfg ServerCfg
}

// ClientCfg contains config which should be common among all rpc client types.
type ClientCfg struct {
	ServerAddress []string
	ServerPort    uint16
	TLS           TLS
	Profile       Profile
	Log           Log
	IsMock        bool
}

// TLS contains config for TLS connections.
type TLS struct {
	Enabled           bool
	PrivateKey        string
	Certificate       string
	RootCAs           []string
	ClientAuthEnabled bool
	ClientRootCAs     []string
}

// PKCS contains config for PKCS keys.
type PKCS struct {
	PrivateKeyType string // PKCS1 or PKCS8
	PrivateKey     string
	Certificate    string
	AppKey         string
	SecretKey      string // unique identification
}

// RunnerConfig ...
type RunnerConfig struct {
	MaxWorkers int
}

// Profile contains configuration for Go pprof profiling.
type Profile struct {
	Enabled bool
	Address string
}

// Log config with rolling backend
// MaxSize is the maximum size in megabytes
// MaxBackups is the maximum number of old log files to retain
// MaxAge is the maximum number of days to retain old log files
type Log struct {
	LogFile    string
	LogLevel   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
}

type RedisCfg struct {
	Enable      bool
	Uri         string
	ConnTimeout int
	CacheTTL    time.Duration
}

type IAM struct {
	CasbinConf string
}

type Token struct {
	ExpiresAt  time.Duration
	PublicKey  string
	PrivateKey string
}

type WdsConfig struct {
	Endpoint string
	Username string
	Password string
	Region   string
	Az       string
}
