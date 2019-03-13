package network

import (
	"fmt"
	"github.com/sirupsen/logrus"
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
	IpRange net.IPNet `json:"ip_range"`
	// 网络驱动名（网络类型）
	Driver string `json:"driver"`
}

func (network *Network) String() string {
	ip, ipNet, _ := net.ParseCIDR(network.IpRange.String())
	return fmt.Sprintf("[%s]%s(ip:%s,range:%s)", network.Driver, network.Name, ip, ipNet)
}

/*
对应一个网络端点，比如容器中会有一个veth和一个loopback
*/
type Endpoint struct {
	Name         string        `json:"name"`
	IpAddress    net.IP        `json:"ip_address"`
	Device       *netlink.Veth `json:"device"`
	Network      *Network      `json:"network"`
	PortMappings []string      `json:"port_mappings"`
}

func (endpoint *Endpoint) GetContainerVethName() string {
	return endpoint.Device.PeerName
}

func (endpoint *Endpoint) GetHostVethName() string {
	return endpoint.Name[:5]
}

/*
如果receiver是指针类型，则接口值必须为指针；如果receiver均为值类型，则接口值可以是指针，也可以是值。
一点规则：有值，未必能取得指针；反之一定可以。
*/
var networkDrivers = map[string]NetworkDriver{
	"bridge": &BridgeNetworkDriver{},
}

func CreateNetwork(driver string, subnet string, name string) (*Network, error) {
	networkDriver, found := networkDrivers[driver]
	if !found {
		return nil, fmt.Errorf("network driver not found: %s", driver)
	}
	return networkDriver.Create(subnet, name)
}

func DeleteNetwork(driver string, name string) error {
	networkDriver, found := networkDrivers[driver]
	if !found {
		return fmt.Errorf("network driver not found: %s", driver)
	}
	network, err := LoadNetwork(driver, name)
	if err != nil {
		return err
	}
	return networkDriver.Delete(network.Name)
}

func LoadNetwork(driver string, name string) (*Network, error) {
	networkDriver, found := networkDrivers[driver]
	if !found {
		return nil, fmt.Errorf("network driver not found: %s", driver)
	}
	return networkDriver.Load(name)
}

func ListNetwork(driver string) ([]*Network, error) {
	networkDriver, found := networkDrivers[driver]
	if !found {
		return nil, fmt.Errorf("network driver not found: %s", driver)
	}
	return networkDriver.List()
}

func ListAllNetwork() ([]*Network, error) {
	var result []*Network
	for _, driver := range networkDrivers {
		networks, err := driver.List()
		if err != nil {
			return nil, err
		}
		result = append(result, networks...)
	}
	return result, nil
}

func Connect(networkDriver string, endpointId string, networkName string, portMappings []string, containerInitPid int) (*Endpoint, error) {
	logrus.Infof("connecting, driver: %s, endpointId: %s, networkName: %s, portMappings: %v, containerInitPid: %d", networkDriver, endpointId, networkName, portMappings, containerInitPid)
	networkDriverInstance, found := networkDrivers[networkDriver]
	if !found {
		return nil, fmt.Errorf("network driver not found: %s", networkDriver)
	}
	return networkDriverInstance.Connect(endpointId, networkName, portMappings, containerInitPid)
}

func Disconnect(endpoint *Endpoint) error {
	logrus.Infof("disconnecting, endpoint: %#v", endpoint)
	networkDriver, found := networkDrivers[endpoint.Network.Driver]
	if !found {
		return fmt.Errorf("network driver not found: %s", endpoint.Network.Driver)
	}
	return networkDriver.Disconnect(endpoint)
}
