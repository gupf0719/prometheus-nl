package registry

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"os"
	"path/filepath"
	"strings"
)

var rm *RegistryManager

type RegistryManager struct {
	rs                map[string]*RegistryService
	reloadCh          chan chan error
	logger            log.Logger
	registryConfigDir string
}

func GetRegistryService(resType string) *RegistryService {
	if resType == "" {
		resType = "rawnode"
	}

	if resType != "rawnode" {
		resType = "app_" + resType
	}

	rs := rm.rs[resType]
	if rs == nil {
		rs = NewRegistryService(rm.logger, rm.reload, rm.registryConfigDir + "/" + resType + ".yaml")
		rm.rs[resType] = rs
		return rs
	}
	return rs
}

func NewRegistryManager(logger log.Logger, reloadCh chan chan error, registryConfigDir string) *RegistryManager {
	s := RegistryManager{
		logger:            logger,
		reloadCh:          reloadCh,
		rs:                make(map[string]*RegistryService),
		registryConfigDir: registryConfigDir,
	}
	s.CreateDir(registryConfigDir)
	s.load(registryConfigDir)
	rm = &s
	return rm
}

func (this *RegistryManager) reload() {
	rc := make(chan error)
	this.reloadCh <- rc
	if err := <-rc; err != nil {
		level.Error(this.logger).Log("msg", "failed to reload config", "err", err)
	}
}

func (this *RegistryManager) CreateDir(path string) {
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(path, os.ModePerm); err != nil {
			level.Error(this.logger).Log("msg", "mkdir failed", "err", err.Error())
		}
	}
}

func (this *RegistryManager) load(path string) {
	files, _ := filepath.Glob(path + "/" + "*.yaml")
	for _, file := range files {
		index := strings.LastIndex(file, "/")
		resType := file[index+1 : len(file)-5]
		this.rs[resType] = NewRegistryService(this.logger, this.reload, file)
	}

}


