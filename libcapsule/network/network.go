package network

import (
	"fmt"
	"github.com/vishvananda/netlink"
	"net"
)

/*
对应一个网段，Driver取值有Bridge
*/
type Network struct {
	// 网络名称
	Name string `json:"name"`
	// 网段
	IpRange *net.IPNet `json:"ip_range"`
	// 网络驱动名（网络类型）
	Driver string `json:"driver"`
}

/*
对应一个网络端点，比如容器中会有一个veth和一个loopback
*/
type Endpoint struct {
	ID           string           `json:"id"`
	IpAddress    net.IP           `json:"ip_address"`
	MacAddress   net.HardwareAddr `json:"mac_address"`
	Device       netlink.Veth     `json:"-"`
	Network      *Network         `json:"network"`
	PortMappings []string         `json:"port_mappings"`
}

/*
如果receiver是指针类型，则接口值必须为指针；如果receiver均为值类型，则接口值可以是指针，也可以是值。
一点规则：有值，未必能取得指针；反之一定可以。
*/
var networkDrivers = map[string]NetworkDriver{
	"bridge":   &BridgeNetworkDriver{},
	"loopback": &LoopbackNetworkDriver{},
}

func CreateNetwork(driver string, subnet string, name string) (*Network, error) {
	networkDriver, found := networkDrivers[driver]
	if !found {
		return nil, fmt.Errorf("network driver not found: %s", driver)
	}
	return networkDriver.Create(subnet, name)
}