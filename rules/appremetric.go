//add by newland
package rules

import (
	"context"
	"fmt"
	"github.com/prometheus/prometheus/pkg/labels"
	"strings"
	"time"
)

type Appremetric struct {
}

func (*Appremetric) RuleFilter(lb *labels.Builder, ctx context.Context, ts time.Time, query QueryFunc) bool {
	pod_name := ""
	for _, l := range lb.Labels() {
		if l.Name == labels.MetricScrapeTagKey {
			return false
		}

		if pod_name == "" && strings.HasSuffix(l.Name, "pod_name") {
			pod_name = l.Value
		}
	}

	if pod_name == "" {
		var metricName string
		for _, l := range lb.Labels() {
			if l.Name == labels.MetricName {
				metricName = l.Value
				break
			}
		}
		fmt.Println(fmt.Sprintf("ERROR: the raw metric [%s] has no label [%s]", metricName, "pod_name"))
		return false
	}

	vector2, err2 := query(ctx, fmt.Sprintf("kube_pod_labels_ch{pod=\"%s\"}", pod_name), ts)
	if err2 == nil && len(vector2) > 0 {
		sample := &vector2[0]
		for _, l := range sample.Metric {
			if strings.HasPrefix(l.Name, "label_") {
				lbname := strings.TrimPrefix(l.Name, "label_")
				if lbname != "pod_template_hash" {
					if strings.HasPrefix(lbname, "k8s_") {
						lbname = strings.TrimPrefix(lbname, "k8s_")
					}
					lb.Set(lbname, l.Value)
				}
			}
		}

	}
	lb.Set(labels.MetricScrapeTagKey, labels.MetricScrapeTagValue)

	return true

}

func init() {
	RegistryRuleFilter("app_define_metric", &Appremetric{})
}
