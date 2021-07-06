package rules

import (
	"context"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/util/strutil"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

type Applabel2ch struct {
}

func (*Applabel2ch) RuleFilter(lb *labels.Builder, ctx context.Context, ts time.Time, query QueryFunc) bool {
	for _, label := range lb.Labels() {
		if strings.HasSuffix(label.Name, "_b64") {
			lbname := strings.TrimSuffix(label.Name, "_b64")
			lbvalue,err := strutil.DecodeByBase64(label.Value)
			if err != nil {
				logrus.Debug(err)
				continue
			}
			lb.Set(lbname, lbvalue)
			lb.Del(label.Name)
		}
	}
	return true
}

func init() {
	RegistryRuleFilter("app_label_2_ch", &Applabel2ch{})
}
