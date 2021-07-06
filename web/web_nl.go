package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prometheus/prometheus/k8scm"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/prometheus/prometheus/promql/parser"
)


var(
	re_for_metric_name = regexp.MustCompile("[a-zA-Z_:][a-zA-Z0-9_:]*")
)

//add by newland
func (h *Handler) pushRecordRules(w http.ResponseWriter, r *http.Request) {
	var msg k8scm.RecordRules
	reqParam, _ := ioutil.ReadAll(r.Body)
	if reqParam == nil {
		http.Error(w, fmt.Sprintf("failed to parse request body: %s", errors.New("request body empty")), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(reqParam, &msg); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse request body: %s", err), http.StatusBadRequest)
		return
	}

	for _, r := range msg {
		if r.Record == "" || r.Expr == "" {
			b, _ := json.Marshal(r)
			http.Error(w, fmt.Sprintf("bad rule: %s", string(b)), http.StatusBadRequest)
			return
		}
		if !re_for_metric_name.MatchString(r.Record) {
			b, _ := json.Marshal(r)
			http.Error(w, fmt.Sprintf("invalid filed[Record] for rule: %s", string(b)), http.StatusBadRequest)
			return
		}
		if _, err := parser.ParseExpr(r.Expr); err != nil {
			b, _ := json.Marshal(r)
			http.Error(w, fmt.Sprintf("bad rule: %s", string(b)), http.StatusBadRequest)
			return
		}

	}

	err := h.rs.UpdRules(msg)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to update rules: %s", err), http.StatusInternalServerError)
	}
}


//add by newland
func (h *Handler) checkPromql(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	q := r.Form.Get("q")
	if q != "" {
		if _, err := parser.ParseExpr(q); err == nil {
			w.WriteHeader(200)
			w.Write([]byte("true"))
			return
		}
		w.WriteHeader(200)
		w.Write([]byte("false"))
		return
	}else {
		http.Error(w, fmt.Sprintf("[q] param cannot empty"), http.StatusBadRequest)
		return
	}
}

func (h *Handler) pushNamedRuleGroups(w http.ResponseWriter, r *http.Request) {
	var msg k8scm.NamedGroups
	reqParam, _ := ioutil.ReadAll(r.Body)
	if reqParam == nil {
		http.Error(w, fmt.Sprintf("failed to parse request body: %s", errors.New("request body empty")), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(reqParam, &msg); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse request body: %s", err), http.StatusBadRequest)
		return
	}

	if msg.Id == "" {
		http.Error(w, fmt.Sprintf("empty filed[Id] for NamedGroups"), http.StatusBadRequest)
		return
	}

	if len(msg.Groups) == 0 {
		http.Error(w, fmt.Sprintf("empty groups for NamedGroups"), http.StatusBadRequest)
		return
	}

	for _, g := range msg.Groups {
		if g.Name == "" {
			http.Error(w, fmt.Sprintf("empty filed[name] for rulegroup"), http.StatusBadRequest)
			return
		}
		if g.Rules == nil || len(g.Rules) == 0 {
			http.Error(w, fmt.Sprintf("empty rules for rulegroup: %s", g.Name), http.StatusBadRequest)
			return
		}
		for _, r := range g.Rules {
			if r.Record == "" || r.Expr == "" {
				b, _ := json.Marshal(r)
				http.Error(w, fmt.Sprintf("empty filed[Record|Expr] for rule: %s", string(b)), http.StatusBadRequest)
				return
			}

			if !re_for_metric_name.MatchString(r.Record) {
				b, _ := json.Marshal(r)
				http.Error(w, fmt.Sprintf("invalid field[Record] for rule: %s", string(b)), http.StatusBadRequest)
				return
			}

			if _, err := parser.ParseExpr(r.Expr); err != nil {
				b, _ := json.Marshal(r)
				http.Error(w, fmt.Sprintf("bad expr for rule: %s", string(b)), http.StatusBadRequest)
				return
			}

		}
	}

	err := h.rs.UpdGroups(&msg)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to update rules: %s", err), http.StatusInternalServerError)
	}
}

func (h *Handler) GetReloadCh() chan chan error {
	return h.reloadCh
}

func (h *Handler) SetRs(rs *k8scm.RecordRuleService) {
	h.rs = rs
}


type TargetRequest struct {
	Labels  map[string]string `json:"labels,omitempty"`
	Targets []string          `json:"targets,omitempty"`
	ResType string  `json:"resType,omitempty"`
}

//add by newland
func (h *Handler) delNamedRuleGroups(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var id string
	if ids, ok := r.Form["id"]; ok {
		if len(ids) > 0 {
			id = ids[0]
		}
	}

	if id == "" {
		http.Error(w, fmt.Sprintf("empty filed[id] for delNamedRuleGroups"), http.StatusBadRequest)
		return
	}

	err := h.rs.DelGroups(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to update rules: %s", err), http.StatusInternalServerError)
	}
}

