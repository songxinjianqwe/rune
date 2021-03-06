package network

type NetworkDriver interface {
	Name() string
	NetworkLabel() string
	Create(subnet string, name string) (*Network, error)
	Load(name string) (*Network, error)
	Delete(name string) error
	Connect(endpointId string, network *Network, portMappings []string, containerInitPid int) (*Endpoint, error)
	Disconnect(endpoint *Endpoint) error
	List() ([]*Network, error)
}
