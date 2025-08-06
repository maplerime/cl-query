/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: database configuration
 *
**/

package conf

type DBConfig struct {
	Url      string
	Type     string
	Debug    bool
	Idle     int
	Open     int
	Lifetime int
}

func (v *DBConfig) GetIdle() int {
	return v.Idle
}

func (v *DBConfig) GetOpen() int {
	return v.Open
}

func (v *DBConfig) GetLifetime() int {
	return v.Lifetime
}

func (v *DBConfig) GetUri() string {
	return v.Url
}

func (v *DBConfig) GetType() string {
	return v.Type
}

func (v *DBConfig) IsDebug() bool {
	return v.Debug
}
