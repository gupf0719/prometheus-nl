package registry

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/prometheus/util"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"reflect"
	"sync"
)

type PromSD struct {
	Labels  map[string]string `json:"labels,omitempty"`
	Targets []string          `json:"targets,omitempty"`
}

type RegistryService struct {
	targetsGroup       *util.BeeMap
	lock               sync.RWMutex
	logger             log.Logger
	registryConfigFile string
	reload func()
}

func NewRegistryService(logger log.Logger, reload func(), registryConfigFile string) *RegistryService {
	s := &RegistryService{
		logger:             logger,
		targetsGroup:       util.NewBeeMap(),
		registryConfigFile: registryConfigFile,
		reload: reload,
	}
	s.load(registryConfigFile)
	return s
}

func (this *RegistryService) UpdPromSDConf(promsd *PromSD) error { //todo

	flush := false
	for _, targetHost := range promsd.Targets {
		newPromsd := &PromSD{
			Labels:  promsd.Labels,
			Targets: []string{targetHost},
		}
		if oldPromSd := this.targetsGroup.Get(targetHost); !reflect.DeepEqual(oldPromSd, newPromsd) {
			this.targetsGroup.Set(targetHost, newPromsd)
			flush = true
		}
	}

	if flush {
		return this.flush()
	}
	return nil
}

func (this *RegistryService)flush() error{
	if err := this.flushData(); err != nil {
		return err
	}

	go this.reload()
	return nil
}

func (this *RegistryService) Delete(promsd *PromSD) error {
	flush := false
	for _, targetHost := range promsd.Targets {
		if oldPromSd := this.targetsGroup.Get(targetHost); oldPromSd != nil {
			this.targetsGroup.Delete(targetHost)
			flush = true
		}
	}

	if flush {
		return this.flush()
	}
	return nil

}

func (this *RegistryService) load(file string){
	if _, err := os.Stat(file); err == nil {
		var promSDs = make([]*PromSD, 0)
		err := ReadInterfaceConfFromFile(file, &promSDs)
		if err != nil {
			level.Error(this.logger).Log("msg", "load conf failed", "file", file, "err", err.Error())
			return
		}

		for _, v := range promSDs {
			this.targetsGroup.Set(v.Targets[0], v)
		}
	}
}

//备份文件？
func (this *RegistryService) flushData() error {
	this.lock.Lock()
	defer this.lock.Unlock()
	m := this.targetsGroup.GetAll()
	promSDs := make([]interface{}, 0, len(m))
	for _, v := range m {
		promSDs = append(promSDs, v)
	}

	bytes, err := yaml.Marshal(promSDs)
	if err != nil {
		level.Error(this.logger).Log("msg", "yaml marshal failed", "err", err.Error())
		return err
	}

	err = ioutil.WriteFile(this.registryConfigFile, bytes, 0666)
	if err != nil {
		level.Error(this.logger).Log("msg", "flush to disk failed", "file", this.registryConfigFile, "err", err.Error())
		return err
	}
	return nil

}

func ReadInterfaceConfFromFile(f string, m interface{}) error {

	b, err := ioutil.ReadFile(f)
	if err != nil {
		return err
	}

	err = yaml.UnmarshalStrict(b, m)
	if err != nil {
		return err
	}

	return nil

}
