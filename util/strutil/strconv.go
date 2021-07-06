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
	"encoding/base64"
	"fmt"
	"hash/crc32"
	"net/url"
	"os"
	"regexp"
	"strconv"
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


//add bynewland
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
//add bynewland
func BinarySearch(strs []string, s string) int {
	lo, hi := 0, len(strs)-1
	for lo <= hi {
		m := (lo + hi) >> 1
		if strs[m] < s {
			lo = m + 1
		} else if strs[m] > s {
			hi = m - 1
		} else {
			return m
		}
	}
	return -1
}
//add bynewland
func DecodeByBase64(labelValue string) (string, error) {
	lbvalue := strings.TrimPrefix(labelValue, "b64")
	lbvalue = strings.TrimSuffix(lbvalue, "0")
	lbvalue = strings.Replace(lbvalue, "-", "+", -1)
	lbvalue = strings.Replace(lbvalue, "_", "/", -1)
	lbvalue = strings.Replace(lbvalue, ".", "=", -1)
	lbvalue = strings.Replace(lbvalue, " ", "+", -1)
	rs, err := base64.StdEncoding.DecodeString(lbvalue)
	if err != nil {
		return "", err
	}
	return string(rs), nil
}
//add bynewland
func GetSplits(s string, n int) int {
	v := int(crc32.ChecksumIEEE([]byte(s)))
	if -v >= 0 {
		v = -v
	}

	return v % n
}
//add bynewland
func GetHashValue(s string) string {
	v := int(crc32.ChecksumIEEE([]byte(s)))
	if -v >= 0 {
		v = -v
	}

	return "r"+strconv.Itoa(v)
}
//add bynewland
func ReplaceVars(s string) string {
	p := regexp.MustCompile(`\$\w+`)
	rs := p.FindAllString(s, -1)
	for _, s0 := range rs {
		s = strings.Replace(s, s0, os.Getenv(strings.TrimPrefix(s0, "$")), -1)
	}

	p = regexp.MustCompile(`\${\w+}`)
	rs = p.FindAllString(s, -1)
	for _, s0 := range rs {
		s = strings.Replace(s, s0, os.Getenv(strings.TrimSuffix(strings.TrimPrefix(s0, "${"), "}")), -1)
	}

	return s
}