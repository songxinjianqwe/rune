package libcapsule

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/songxinjianqwe/capsule/libcapsule/cgroups"
	"github.com/songxinjianqwe/capsule/libcapsule/configs"
	"github.com/songxinjianqwe/capsule/libcapsule/constant"
	"github.com/songxinjianqwe/capsule/libcapsule/network"
	"github.com/songxinjianqwe/capsule/libcapsule/util/exception"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
)

func NewFactory(runtimeRoot string, init bool) (Factory, error) {
	//logrus.Infof("new container factory ...")
	if runtimeRoot == "" {
		runtimeRoot = constant.DefaultRuntimeRoot
	}
	if init {
		if _, err := os.Stat(runtimeRoot); err != nil {
			if os.IsNotExist(err) {
				logrus.Infof("mkdir DefaultRuntimeRoot if not exists: %s", runtimeRoot)
				if err := os.MkdirAll(runtimeRoot, 0700); err != nil {
					return nil, exception.NewGenericError(err, exception.FactoryNewError)
				}
			} else {
				return nil, exception.NewGenericError(err, exception.FactoryNewError)
			}
		}
	}
	factory := &LinuxContainerFactory{root: runtimeRoot}
	if err := network.InitNetworkDrivers(runtimeRoot); err != nil {
		return nil, err
	}
	return factory, nil
}

type LinuxContainerFactory struct {
	root string
}

func (factory *LinuxContainerFactory) GetRuntimeRoot() string {
	return factory.root
}

func (factory *LinuxContainerFactory) Create(id string, config *configs.ContainerConfig) (Container, error) {
	logrus.Infof("container factory creating container: %s", id)
	containerRoot := filepath.Join(factory.root, constant.ContainerDir, id)
	// 如果该目录已经存在(err == nil)，则报错；如果有其他错误(忽略目录不存在的错，我们希望目录不存在)，则报错
	if _, err := os.Stat(containerRoot); err == nil {
		return nil, exception.NewGenericError(fmt.Errorf("container with id exists: %v", id), exception.ContainerIdExistsError)
	} else if !os.IsNotExist(err) {
		return nil, exception.NewGenericError(err, exception.ContainerLoadError)
	}
	logrus.Infof("mkdir root: %s", containerRoot)
	if err := os.MkdirAll(containerRoot, 0644); err != nil {
		return nil, exception.NewGenericError(err, exception.ContainerRootCreateError)
	}
	container := &LinuxContainer{
		id:            id,
		runtimeRoot:   factory.root,
		containerRoot: containerRoot,
		config:        *config,
		cgroupManager: cgroups.NewCroupManager(id, make(map[string]string)),
	}
	container.statusBehavior = &StoppedStatusBehavior{c: container}
	logrus.Infof("create container complete, container: %#v", container)
	return container, nil
}

func (factory *LinuxContainerFactory) Exists(id string) bool {
	containerRoot := filepath.Join(factory.root, constant.ContainerDir, id)
	_, err := factory.loadContainerState(containerRoot, id)
	return err == nil
}

func (factory *LinuxContainerFactory) Load(id string) (Container, error) {
	containerRoot := filepath.Join(factory.root, constant.ContainerDir, id)
	state, err := factory.loadContainerState(containerRoot, id)
	if err != nil {
		return nil, err
	}
	loadedNetwork, err := network.LoadNetwork(state.Endpoint.Network.Driver, state.Endpoint.Network.Name)
	if err != nil {
		return nil, err
	}
	state.Endpoint.Network = loadedNetwork
	container := &LinuxContainer{
		id:            id,
		createdTime:   state.Created,
		runtimeRoot:   factory.root,
		containerRoot: containerRoot,
		config:        state.Config,
		endpoint:      state.Endpoint,
		cgroupManager: cgroups.NewCroupManager(id, state.CgroupPaths),
	}
	container.parentProcess = NewParentNoChildProcess(state.InitProcessPid, state.InitProcessStartTime, container)
	detectedStatus, err := container.detectContainerStatus()
	if err != nil {
		return nil, err
	}
	// 目前的状态
	container.statusBehavior, err = NewContainerStatusBehavior(detectedStatus, container)
	if err != nil {
		return nil, err
	}
	return container, nil
}

func (factory *LinuxContainerFactory) StartInitialization() error {
	logrus.Infof("capsule init/StartInitialization...")
	defer func() {
		if e := recover(); e != nil {
			logrus.Errorf("panic from initialization: %v, %v", e, string(debug.Stack()))
		}
	}()
	configPipeEnv := os.Getenv(constant.EnvConfigPipe)
	initPipeFd, err := strconv.Atoi(configPipeEnv)
	logrus.WithField("init", true).Infof("got config pipe env: %d", initPipeFd)
	if err != nil {
		return exception.NewGenericErrorWithContext(err, exception.EnvError, "converting EnvConfigPipe to int")
	}
	initializerType := InitializerType(os.Getenv(constant.EnvInitializerType))
	logrus.WithField("init", true).Infof("got initializer type: %s", initializerType)

	// 读取config
	configPipe := os.NewFile(uintptr(initPipeFd), "configPipe")
	logrus.WithField("init", true).Infof("open child pipe: %#v", configPipe)
	logrus.WithField("init", true).Infof("starting to read init config from child pipe")
	bytes, err := ioutil.ReadAll(configPipe)
	if err != nil {
		logrus.WithField("init", true).Errorf("read init config failed: %s", err.Error())
		return exception.NewGenericErrorWithContext(err, exception.PipeError, "reading init config from configPipe")
	}
	// child 读完就关
	if err = configPipe.Close(); err != nil {
		logrus.Errorf("closing parent pipe failed: %s", err.Error())
	}
	logrus.Infof("read init config complete, unmarshal bytes")
	initConfig := &InitExecConfig{}
	if err = json.Unmarshal(bytes, initConfig); err != nil {
		return exception.NewGenericErrorWithContext(err, exception.PipeError, "unmarshal init config")
	}
	logrus.WithField("init", true).Infof("read init config from child pipe: %#v", initConfig)

	// 环境变量设置
	if err := populateProcessEnvironment(initConfig.ProcessConfig.Env); err != nil {
		return exception.NewGenericErrorWithContext(err, exception.EnvError, "populating environment variables")
	}

	// 创建Initializer
	initializer, err := NewInitializer(initializerType, initConfig, configPipe, factory.root)
	if err != nil {
		return exception.NewGenericErrorWithContext(err, exception.InitializerCreateError, "creating initializer")
	}
	logrus.WithField("init", true).Infof("created initializer:%#v", initializer)

	// 正式开始初始化
	if err := initializer.Init(); err != nil {
		return exception.NewGenericErrorWithContext(err, exception.InitializerRunError, "executing init command")
	}
	return nil
}

// populateProcessEnvironment loads the provided environment variables into the
// current processes's environment.
func populateProcessEnvironment(env []string) error {
	for _, pair := range env {
		splits := strings.SplitN(pair, "=", 2)
		if len(splits) < 2 {
			return fmt.Errorf("invalid environment '%v'", pair)
		}
		logrus.WithField("init", true).Infof("set env: key:%s, value:%s", splits[0], splits[1])
		if err := os.Setenv(splits[0], splits[1]); err != nil {
			return err
		}
	}
	return nil
}

func (factory *LinuxContainerFactory) loadContainerState(containerRoot, id string) (*StateStorage, error) {
	stateFilePath := filepath.Join(containerRoot, constant.StateFilename)
	f, err := os.Open(stateFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, exception.NewGenericError(fmt.Errorf("container %s does not exist", id), exception.ContainerNotExistsError)
		}
		return nil, exception.NewGenericError(err, exception.ContainerLoadError)
	}
	defer f.Close()
	var state *StateStorage
	if err := json.NewDecoder(f).Decode(&state); err != nil {
		return nil, exception.NewGenericError(err, exception.ContainerLoadError)
	}
	return state, nil
}
