/*
newland
record rule
 */

package k8scm

import (
	"context"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/prometheus/common"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/rulefmt"
	"github.com/prometheus/prometheus/util/strutil"
	"gopkg.in/yaml.v2"
	yaml3 "gopkg.in/yaml.v3"
	"os"
	"time"
)

var (
	prometheusSdCm string
)

func init(){
	if cm := os.Getenv("PROMETHEUS_SD_CM"); cm != ""{
		prometheusSdCm = cm
	}else {
		prometheusSdCm = "prometheus-rules-config"
	}
}

type RecordRule struct {
	Record  string            `json:"record,omitempty"`
	Expr    string            `json:"expr,omitempty"`
	Filters []string          `json:"filters,omitempty"`
	Labels  map[string]string `json:"labels,omitempty"`
}

type RecordRules []RecordRule

type CmRecordRules struct {
	Tags   map[string]string `json:"tags,omitempty",yaml:"tags,omitempty"`
	Groups []Group           `json:"groups,omitempty",yaml:"groups,omitempty"`
}

type Group struct {
	Name  string      `json:"name,omitempty",yaml:"name,omitempty"`
	Rules RecordRules `json:"rules,omitempty",yaml:"rules,omitempty"`
}

type NamedGroups struct {
	Id     string            `json:"id,omitempty",yaml:"id,omitempty"`
	Tags   map[string]string `json:"tags,omitempty",yaml:"tags,omitempty"`
	Groups []Group           `json:"groups,omitempty",yaml:"groups,omitempty"`
}

type RecordRuleService struct {
	CmService
	logger   log.Logger
	reloadCh chan chan error
}

func NewRecordRuleService(logger log.Logger, reloadCh chan chan error) *RecordRuleService {
	s := RecordRuleService{
		logger:   common.GetLogger(),
		reloadCh: reloadCh,
	}
	s.CmService.logger = common.GetLogger()
	return &s
}

//ruleId: 规则id(集群id, 应用id(单元id)等)
func (this *RecordRuleService) UpdRules(rules RecordRules) error { //todo
	if rules == nil || len(rules) == 0 {
		return fmt.Errorf("empty RecordRules.")
	}

	n := 100
	groups := make([]Group, n)
	for _, r := range rules {
		code := strutil.GetSplits(r.Record, n)
		if groups[code].Name == "" {
			groups[code].Name = fmt.Sprintf("meta_metrics_%v", code)
			groups[code].Rules = make(RecordRules, 0)
		}

		groups[code].Rules = append(groups[code].Rules, r)
	}

	newGroups := make([]Group, 0)
	for _, g := range groups {
		if g.Name != "" {
			newGroups = append(newGroups, g)
		}
	}

	CmRecordRules := CmRecordRules{
		Groups: newGroups,
	}

	result, err := yaml.Marshal(CmRecordRules)
	if err != nil {
		level.Error(this.logger).Log("msg", "convert obj to yaml failed", "err", err)
		return err
	}

	oldVersion, err := this.upd(prometheusSdCm, map[string]string{"recordRules.yml": string(result)})
	if err != nil {
		level.Error(this.logger).Log("msg", "upd config", "err", err)
		return err
	}

	this.reload(oldVersion)

	return nil
}

//ruleId: 规则id(集群id, 应用id(单元id)等)
func (this *RecordRuleService) UpdGroups(namedGroups *NamedGroups) error { //todo
	if namedGroups == nil || len(namedGroups.Groups) == 0 {
		return fmt.Errorf("empty RecordRules.")
	}
	for n, g := range namedGroups.Groups {
		rules := g.Rules
		for i, r := range rules {
			if r.Labels == nil {
				r.Labels = make(map[string]string, 1)
			}
			//mtype: sys
			r.Labels[labels.MetricScrapeTagKey] = labels.MetricScrapeTagValue
			rules[i] = r
		}
		g.Rules = rules
		namedGroups.Groups[n] = g
	}

	CmRecordRules := CmRecordRules{
		Tags:   namedGroups.Tags,
		Groups: namedGroups.Groups,
	}

	result, err := yaml.Marshal(CmRecordRules)
	if err != nil {
		level.Error(this.logger).Log("msg", "convert obj to yaml failed", "err", err)
		return err
	}

	oldVersion, err := this.upd(prometheusSdCm, map[string]string{namedGroups.Id: string(result)})
	if err != nil {
		level.Error(this.logger).Log("msg", "upd config", "err", err)
		return err
	}

	this.reload(oldVersion)

	return nil
}

//ruleId: 规则id(集群id, 应用id(单元id)等)
func (this *RecordRuleService) DelGroups(namedGroupsId string) error { //todo
	if namedGroupsId == "" {
		return fmt.Errorf("empty namedGroupsId.")
	}

	oldVersion, err := this.updForDel(prometheusSdCm, []string{namedGroupsId})
	if err != nil {
		level.Error(this.logger).Log("msg", "upd config", "err", err)
		return err
	}

	this.reload(oldVersion)

	return nil
}

//configmap是异步更新，这里reload的时候，可能配置未生效
func (this *RecordRuleService) reload(oldVersion string) {
	go func() {
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Minute)

		ok := make(chan bool)
		go this.checkUpdFinished(ctx, prometheusSdCm, oldVersion, ok)

		select {
		case finished := <-ok:
			if finished {
				rc := make(chan error)
				this.reloadCh <- rc
				if err := <-rc; err != nil {
					level.Error(this.logger).Log("msg", "failed to reload config", "err", err)
				}
			} else {
				level.Error(this.logger).Log("msg", "failed to reload rules")
			}
			return
		case <-ctx.Done():
			level.Error(this.logger).Log("msg", " Timeout for reload rules in 10m")
			return
		}

	}()
}

func (this *RecordRuleService) ParseFromConfigmap() (*rulefmt.RuleGroups, []error) {
	datas, err := this.get(prometheusSdCm)
	if err != nil {
		return nil, []error{err}
	}

	rulegroups := make([]rulefmt.RuleGroup, 0)
	errors := make([]error, 0)
	for key, data := range datas {
		rulegroups0, errors0 := this.ParseDatas(data)
		rulegroups = append(rulegroups, rulegroups0...)
		errors = append(errors, errors0...)
		level.Info(this.logger).Log("msg", "parse to rule end.", "rulefilename", key)
	}

	rgs := &rulefmt.RuleGroups{
		Groups: rulegroups,
	}

	if len(errors) == 0 {
		return rgs, nil
	}

	return rgs, errors
}

func (this *RecordRuleService) ParseDatas(data string) ([]rulefmt.RuleGroup, []error) {

	var recordrules CmRecordRules
	err := yaml.Unmarshal([]byte(data), &recordrules)
	if err != nil {
		return nil, []error{err}
	}


	rulegroups := make([]rulefmt.RuleGroup, 0)
	errs := make([]error, 0)

	for _, group := range recordrules.Groups {
		rules := make([]rulefmt.RuleNode, 0, len(group.Rules))
		for _, meta := range group.Rules {
			var record yaml3.Node
			var expr yaml3.Node
			record.SetString(meta.Record)
			expr.SetString(meta.Expr)
			rules = append(rules, rulefmt.RuleNode{
				Record:  record,
				Expr:    expr,
				Filters: meta.Filters,
				Labels:  meta.Labels,
			})
		}
		rg := rulefmt.RuleGroup{}
		rg.Name = group.Name
		rg.Rules = rules

		/*rgs := &rulefmt.RuleGroups{
			Groups: []rulefmt.RuleGroup{rg},
		}*/

		for _,node := range rg.Rules{
			errTmp := node.Validate()
			if len(errTmp) != 0 {
				errs = append(errs, node.Validate()[0].Error())
			}
		}

		//errs = append(errs, rg.Rules ...)
		rulegroups = append(rulegroups, rg)
	}

	return rulegroups, errs

}
