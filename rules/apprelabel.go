package rules

import (
	"context"
	"fmt"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/util/strutil"
	"strings"
	"time"
)

type Apprelabel struct {
}

func (*Apprelabel) RuleFilter(lb *labels.Builder, ctx context.Context, ts time.Time, query QueryFunc) bool {
	pod_name := ""
	for _, l := range lb.Labels() {
		if strings.HasSuffix(l.Name, "pod_name") {
			pod_name = l.Value
			break
		}
	}

	if pod_name != "" {

		vector2, err2 := query(ctx, fmt.Sprintf("kube_pod_labels{pod=\"%s\"}", pod_name), ts)
		if err2 == nil && len(vector2) > 0 {
			sample := &vector2[0]
			for _, l := range sample.Metric {
				if strings.HasPrefix(l.Name, "label_") {
					lbname := strings.TrimPrefix(l.Name, "label_")
					if lbname != "pod_template_hash" && lbname != "controller_revision_hash" {
						if strings.HasPrefix(lbname, "k8s_") {
							lbname = strings.TrimPrefix(lbname, "k8s_")
						}
						lb.Set(lbname, l.Value)
					}
				}
			}

			lblabels := lb.Labels()
			if lblabels.Get("appid") == "" {
				if lv := sample.Metric.Get("label_pod_template_hash"); lv != "" {
					lb.Set("appid", strutil.GetHashValue(lv+lblabels.Get("clusterId")))
				} else if lv := sample.Metric.Get("label_controller_revision_hash"); lv != "" {
					lb.Set("appid", strutil.GetHashValue(lv))
				}

			}
		}

	}

	return true

}

func init() {
	RegistryRuleFilter("app_append_labels", &Apprelabel{})
}
