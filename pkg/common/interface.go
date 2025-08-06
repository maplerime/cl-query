/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    jack@raksmart.com - Initial implementation
 *
 *
 * Purpose:
 *
**/

package common

type VlanInfo struct {
	Device        string          `json:"device"`
	Vlan          int64           `json:"vlan"`
	Gateway       string          `json:"gateway"`
	Router        int64           `json:"router"`
	PublicLink    int64           `json:"public_link"`
	Inbound       int32           `json:"inbound"`
	Outbound      int32           `json:"outbound"`
	AllowSpoofing bool            `json:"allow_spoofing"`
	IpAddr        string          `json:"ip_address"`
	MacAddr       string          `json:"mac_address"`
	SecRules      []*SecurityData `json:"security"`
}

type SecurityData struct {
	Secgroup    int64
	RemoteIp    string `json:"remote_ip"`
	RemoteGroup int64  `json:"remote_group"`
	Direction   string `json:"direction"`
	IpVersion   string `json:"ip_version"`
	Protocol    string `json:"protocol"`
	PortMin     int32  `json:"port_min"`
	PortMax     int32  `json:"port_max"`
}

type NetworkRoute struct {
	Network string `json:"network"`
	Netmask string `json:"netmask"`
	Gateway string `json:"gateway"`
}
