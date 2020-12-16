// Copyright 2013 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package strutil

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

// TableLinkForExpression creates an escaped relative link to the table view of
// the provided expression.
func TableLinkForExpression(expr string) string {
	escapedExpression := url.QueryEscape(expr)
	return fmt.Sprintf("/graph?g0.expr=%s&g0.tab=1", escapedExpression)
}

// GraphLinkForExpression creates an escaped relative link to the graph view of
// the provided expression.
func GraphLinkForExpression(expr string) string {
	escapedExpression := url.QueryEscape(expr)
	return fmt.Sprintf("/graph?g0.expr=%s&g0.tab=0", escapedExpression)
}

// SanitizeLabelName replaces anything that doesn't match
// client_label.LabelNameRE with an underscore.
func SanitizeLabelName(name string) string {
	return invalidLabelCharRE.ReplaceAllString(name, "_")
}

// newland
func SplitString(s string, split string) (s1 string, s2 string) {
	ss := strings.Split(s, split)
	s1 = strings.TrimSpace(ss[0])
	if len(ss) == 1 {
		s2 = s1
		return
	}

	s2 = strings.TrimSpace(ss[1])
	if s2 == "" {
		s2 = s1
	}
	return
}