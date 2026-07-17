/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: Resource adapter factory
 *
**/

package resources

import (
	"github.com/maplerime/cl-query/pkg/common"

	"github.com/maplerime/cl-query/pkg/api/resources/adapters"
)

// AdapterFactory 适配器工厂
type AdapterFactory struct {
	adapters map[string]func() adapters.ResourceAdapter
}

// NewAdapterFactory 创建适配器工厂
func NewAdapterFactory() *AdapterFactory {
	factory := &AdapterFactory{
		adapters: make(map[string]func() adapters.ResourceAdapter),
	}
	factory.Register("instance", func() adapters.ResourceAdapter {
		return adapters.NewInstanceAdapter()
	})
	factory.Register("volume", func() adapters.ResourceAdapter {
		return adapters.NewVolumeAdapter()
	})
	factory.Register("security_group", func() adapters.ResourceAdapter {
		return adapters.NewSecurityGroupAdapter()
	})
	factory.Register("security_rule", func() adapters.ResourceAdapter {
		return adapters.NewSecruleAdapter()
	})
	factory.Register("interface", func() adapters.ResourceAdapter {
		return adapters.NewInterfaceAdapter()
	})
	factory.Register("image", func() adapters.ResourceAdapter {
		return adapters.NewImageAdapter()
	})
	factory.Register("vpc", func() adapters.ResourceAdapter {
		return adapters.NewVPCAdapter()
	})
	factory.Register("ip_group", func() adapters.ResourceAdapter {
		return adapters.NewIpGroupAdapter()
	})
	factory.Register("floating_ip", func() adapters.ResourceAdapter {
		return adapters.NewFloatingIPAdapter()
	})
	factory.Register("subnet", func() adapters.ResourceAdapter {
		return adapters.NewSubnetAdapter()
	})
	factory.Register("ip_address", func() adapters.ResourceAdapter {
		return adapters.NewAddressAdapter()
	})
	factory.Register("dictionary", func() adapters.ResourceAdapter {
		return adapters.NewDictionaryAdapter()
	})
	factory.Register("hyper", func() adapters.ResourceAdapter {
		return adapters.NewHyperAdapter()
	})
	factory.Register("zone", func() adapters.ResourceAdapter {
		return adapters.NewZoneAdapter()
	})
	factory.Register("migration", func() adapters.ResourceAdapter {
		return adapters.NewMigrationAdapter()
	})
	return factory
}

// Register 注册适配器
func (f *AdapterFactory) Register(resourceType string, creator func() adapters.ResourceAdapter) {
	f.adapters[resourceType] = creator
}

// CreateAdapter 创建适配器
func (f *AdapterFactory) CreateAdapter(resourceType string) (adapters.ResourceAdapter, error) {
	creator, exists := f.adapters[resourceType]
	if !exists {
		return nil, common.NewCLError(common.ErrInvalidParameter, "unsupported resource type", nil)
	}
	return creator(), nil
}

// GetSupportedTypes 获取支持的资源类型
func (f *AdapterFactory) GetSupportedTypes() []string {
	types := make([]string, 0, len(f.adapters))
	for resourceType := range f.adapters {
		types = append(types, resourceType)
	}
	return types
}

// DefaultFactory 全局工厂实例
var DefaultFactory = NewAdapterFactory()
