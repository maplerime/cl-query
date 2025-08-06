/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: Channel management service
 *
**/

package common

import (
	xmeta "github.com/maplerime/cl-query/utils/metadata"
)

// BuildNumber ...
const (
	ProgramName = "cl-query"
)

// service metadata
type Metadata struct {
	SVCID      string         `json:"svc_id"`
	APIVersion *xmeta.Version `json:"version"`
	// ...
	// add more fields as needed
}

var metadata *Metadata

// GetSvcMetadata ...
func GetSvcMetadata() *Metadata {
	if metadata == nil {
		version := xmeta.ParseVersion(VersionFile)
		version.ProgramName = ProgramName
		metadata = &Metadata{
			APIVersion: version,
		}
	}
	if Config != nil {
		metadata.SVCID = Config.RPCSvc.SVCID
	}
	return metadata
}

func SetSvcMetadata(m *Metadata) {
	metadata = m
}
