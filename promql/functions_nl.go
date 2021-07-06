package promql

import (
	"fmt"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/prometheus/prometheus/util/strutil"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	TAG_FOR_VALUE = "__v__"
	TAG_VALUE     = "_v"
	SPLIT_SYMBOL_1 = "|"
)


type varType int
type symbolType int

const (
	STR varType = iota
	NOSTR
)

const (
	EQ symbolType = iota
	NE
	GT
	LT
	GE
	LE
)

type SortVector struct {
	Vector
	SortKey string
}

func (m SortVector) Len() int           { return len(m.Vector) }
func (m SortVector) Less(i, j int) bool {
	return m.Vector[i].Metric.Get(m.SortKey) < m.Vector[j].Metric.Get(m.SortKey)
}
func (m SortVector) Swap(i, j int)      { m.Vector[i], m.Vector[j] = m.Vector[j], m.Vector[i] }

//add by newland
type Points []Point

func (ls Points) Len() int           { return len(ls) }
func (ls Points) Swap(i, j int)      { ls[i], ls[j] = ls[j], ls[i] }
func (ls Points) Less(i, j int) bool { return ls[i].V < ls[j].V }


var RE_TERNARY_OPERATOR = regexp.MustCompile(`^(\w+)([=!><]=|[<>])('[.\w]*'|[.\w]+)\?('\w*'|\w+):('\w*'|\w+)$`)
var RE_STR = regexp.MustCompile(`^'[.\w]*'$`)

func Parse(s string) (*ternaryOperator, error) {
	bool := RE_TERNARY_OPERATOR.MatchString(s)
	if !bool {
		return nil, fmt.Errorf("Not match for ternary operator.")
	}

	ss := RE_TERNARY_OPERATOR.FindStringSubmatch(s)
	topr := &ternaryOperator{
		OprData1:   NewOprData(ss[1]),
		OprData2:   NewOprData(ss[3]),
		OprData3:   NewOprData(ss[4]),
		OprData4:   NewOprData(ss[5]),
		SymbolType: GetSymbolType(ss[2]),
	}

	return topr, nil
}

type ternaryOperator struct {
	OprData1   *OprData
	OprData2   *OprData
	OprData3   *OprData
	OprData4   *OprData
	SymbolType symbolType
}

type OprData struct {
	value   string
	varType varType
}

func NewOprData(v string) *OprData {
	bool := RE_STR.MatchString(v)
	if bool {
		return &OprData{
			value:   strings.Trim(v, "'"),
			varType: STR,
		}
	}

	return &OprData{
		value:   v,
		varType: NOSTR,
	}
}

func GetSymbolType(s string) symbolType {
	switch s {
	case "==":
		return EQ
	case "!=":
		return NE
	case ">=":
		return GE
	case "<=":
		return LE
	case ">":
		return GT
	case "<":
		return LT
	default:
		return -1
	}
}

func parseTernaryOperatorWithSample(s *Sample, expr *ternaryOperator) string {
	var oprdata1 = parseValue(s, expr.OprData1)
	var oprdata2 = parseValue(s, expr.OprData2)
	var oprdata3 = parseValue(s, expr.OprData3)
	var oprdata4 = parseValue(s, expr.OprData4)

	oprval := false
	switch expr.SymbolType {
	case EQ:
		oprval = oprdata1 == oprdata2
	case NE:
		oprval = oprdata1 != oprdata2
	case GE:
		oprval = oprdata1 >= oprdata2
	case LE:
		oprval = oprdata1 <= oprdata2
	case GT:
		oprval = oprdata1 > oprdata2
	case LT:
		oprval = oprdata1 < oprdata2
	}

	if oprval {
		return oprdata3
	}
	return oprdata4
}

func parseValue(s *Sample, d *OprData) string {
	var oprdata1 string
	if d.varType == NOSTR {
		if d.value == TAG_FOR_VALUE+"0" {
			oprdata1 = strconv.FormatFloat(s.V, 'f', -1, 64)
		} else {
			oprdata1, _ = s.Metric.BinarySearch(d.value)
		}
	} else {
		oprdata1 = d.value
	}
	return oprdata1
}

//funcLabelsAppendTagsByTernary
func funcLabelsAppendTagsByTernary(vals []parser.Value, args parser.Expressions, enh *EvalNodeHelper) Vector {
	leng := len(args) - 1
	var (
		vector       = vals[0].(Vector)
		targetKeys   = make([]string, leng/2)
		targetValues = make([]*ternaryOperator, leng/2)
	)

	if enh.Dmn == nil {
		enh.Dmn = make(map[uint64]labels.Labels, len(enh.Out))
	}

	for i := 1; i < len(args); i++ {
		src := stringFromArg(args[i])
		if i%2 == 0 {
			topr, err := Parse(src)
			if err != nil {
				panic(fmt.Errorf("invalid ternary operator in labels_append_ternary(): %s", src))
			}
			targetValues[(i-2)/2] = topr
		} else {
			if !model.LabelName(src).IsValid() {
				panic(fmt.Errorf("invalid source label name in labels_append_ternary(): %s", src))
			}
			targetKeys[i/2] = src
		}
	}

	if len(targetKeys) != len(targetValues) {
		panic("invalid params in labels_append_tags()")
	}

	for _, el := range vector {
		newlabels := el.Metric
		for i, targetKey := range targetKeys {
			var targetValue = targetValues[i]
			sort.Sort(newlabels)
			var newValue = parseTernaryOperatorWithSample(&el, targetValue)
			if targetKey == TAG_FOR_VALUE+"0" {
				floatV, err := strconv.ParseFloat(newValue, 64)
				if err == nil {
					el.V = floatV
				}
				continue
			}
			_, index := newlabels.BinarySearch(targetKey)
			if index != -1 {
				newlabels[index] = labels.Label{Name: targetKey, Value: newValue}
			} else {
				newlabels = append(newlabels, labels.Label{Name: targetKey, Value: newValue})
			}
			el.Metric = newlabels
		}

		//el.Metric = newlabels
		sort.Sort(el.Metric)
		enh.Out = append(enh.Out, el)
	}
	return enh.Out
}

//add by newland
func funcLabelsAppendTags(vals []parser.Value, args parser.Expressions, enh *EvalNodeHelper) Vector {
	leng := len(args) - 1
	var (
		vector       = vals[0].(Vector)
		targetKeys   = make([]string, leng/2)
		targetValues = make([]string, leng/2)
	)

	if enh.Dmn == nil {
		enh.Dmn = make(map[uint64]labels.Labels, len(enh.Out))
	}

	for i := 1; i < len(args); i++ {
		src := stringFromArg(args[i])
		if !model.LabelName(src).IsValid() {
			panic(fmt.Errorf("invalid source label name in labels_append_tags(): %s", src))
		}

		if i%2 == 0 {
			targetValues[(i-2)/2] = src
		} else {
			targetKeys[i/2] = src
		}
	}

	if len(targetKeys) != len(targetValues) {
		panic("invalid params in labels_append_tags()")
	}

	for _, el := range vector {
		newlabels := el.Metric
		for i, targetKey := range targetKeys {
			var targetValue = targetValues[i]
			var exist bool
			for j, l := range newlabels {
				if l.Name == targetKey {
					if l.Value == "" {
						l.Value = targetValue
						newlabels[j] = l
					}
					exist = true
					break
				}
			}
			if !exist {
				newlabels = append(newlabels, labels.Label{Name: targetKey, Value: targetValue})
			}
		}

		el.Metric = newlabels
		sort.Sort(el.Metric)
		enh.Out = append(enh.Out, el)
	}
	return enh.Out
}

//add by newland
func funcLabelsRename(vals []parser.Value, args parser.Expressions, enh *EvalNodeHelper) Vector {
	leng := len(args) - 1
	var (
		vector     = vals[0].(Vector)
		targetKeys = make([]string, leng/2)
		newKeys    = make([]string, leng/2)
	)

	if enh.Dmn == nil {
		enh.Dmn = make(map[uint64]labels.Labels, len(enh.Out))
	}

	for i := 1; i < len(args); i++ {
		src := stringFromArg(args[i])
		if !model.LabelName(src).IsValid() {
			panic(fmt.Errorf("invalid source label name in labels_invert(): %s", src))
		}

		if i%2 == 0 {
			newKeys[(i-2)/2] = src
		} else {
			targetKeys[i/2] = src
		}
	}

	if len(targetKeys) != len(newKeys) {
		panic("invalid params in labels_append()")
	}

	for _, el := range vector {
		newlabels := el.Metric
		for i, l := range newlabels {
			for j, targetKey := range targetKeys {
				if l.Name == targetKey {
					l.Name = newKeys[j]
					newlabels[i] = l
					break
				}
			}
		}
		el.Metric = newlabels

		sort.Sort(el.Metric)
		enh.Out = append(enh.Out, el)

	}
	return enh.Out
}

//add by newland
func funcLabelsSelect(vals []parser.Value, args parser.Expressions, enh *EvalNodeHelper) Vector {
	var (
		vector    = vals[0].(Vector)
		srcLabels = make([]string, len(args)-1)
	)

	if enh.Dmn == nil {
		enh.Dmn = make(map[uint64]labels.Labels, len(enh.Out))
	}

	for i := 1; i < len(args); i++ {
		src := stringFromArg((args[i]))
		if !model.LabelName(src).IsValid() {
			panic(fmt.Errorf("invalid source label name in labels_invert(): %s", src))
		}
		srcLabels[i-1] = src
	}

	for _, el := range vector {
		newlabels := make(labels.Labels, 0)
		for _, label := range srcLabels {
			if lv := el.Metric.Get(label); lv != "" {
				newlabels = append(newlabels, labels.Label{label, lv})
			}
		}
		if len(newlabels) > 0 {
			el.Metric = newlabels
			sort.Sort(el.Metric)
			enh.Out = append(enh.Out, el)
			continue
		}
	}
	return enh.Out
}

//add by newland
func funcSelect1(vals []parser.Value, args parser.Expressions, enh *EvalNodeHelper) Vector {
	leng := len(args) - 1
	var (
		vector = vals[0].(Vector)
		//srcLabels = make([]string, len(args)-1)
		targetKeys = make([]string, leng)
		newKeys    = make([]string, leng)
	)

	if enh.Dmn == nil {
		enh.Dmn = make(map[uint64]labels.Labels, len(enh.Out))
	}

	for i := 1; i < len(args); i++ {
		targetkey, newkey := strutil.SplitString(stringFromArg(args[i]), ":")

		if !model.LabelName(targetkey).IsValid() {
			panic(fmt.Errorf("invalid source label name in select1(): %s", targetkey))
		}
		if !model.LabelName(newkey).IsValid() {
			panic(fmt.Errorf("invalid source label name in select1(): %s", newkey))
		}
		targetKeys[i-1], newKeys[i-1] = targetkey, newkey
	}

	for _, el := range vector {
		newlabels := make(labels.Labels, 0)
		for i, label := range targetKeys {
			if lv := el.Metric.Get(label); lv != "" {
				newlabels = append(newlabels, labels.Label{newKeys[i], lv})
				continue
			}

			if label == TAG_FOR_VALUE+"0" {
				newlabels = append(newlabels, labels.Label{newKeys[i], strconv.FormatFloat(el.V, 'f', -1, 64)})
			}
		}
		if len(newlabels) > 0 {
			el.Metric = newlabels
			sort.Sort(el.Metric)
			enh.Out = append(enh.Out, el)
			continue
		}
	}
	return enh.Out
}

//add by newland
func funcSelect2(vals []parser.Value, args parser.Expressions, enh *EvalNodeHelper) Vector {
	leng := len(args) - 3
	var (
		vector     = vals[1].(Vector)
		vector1    = vals[2].(Vector)
		targetKeys = make([]string, leng)
		newKeys    = make([]string, leng)
	)
	srcMatchKey, destMatchKey := strutil.SplitString(stringFromArg(args[0]), SPLIT_SYMBOL_1)
	for i := 3; i < len(args); i++ {
		targetkey, newkey := strutil.SplitString(stringFromArg(args[i]), ":")

		if !model.LabelName(targetkey).IsValid() {
			panic(fmt.Errorf("invalid source label name in select_in(): %s", targetkey))
		}
		if !model.LabelName(newkey).IsValid() {
			panic(fmt.Errorf("invalid source label name in select_in(): %s", newkey))
		}
		targetKeys[i-3], newKeys[i-3] = targetkey, newkey
	}

	if enh.Dmn == nil {
		enh.Dmn = make(map[uint64]labels.Labels, len(enh.Out))
	}

	var srcTargetKey, srcNewkey string
	for i, targetKey := range targetKeys {
		if targetKey == TAG_FOR_VALUE+"0" {
			srcTargetKey = targetKey
			srcNewkey = newKeys[i]
		}
	}

	sort.Sort(SortVector{vector, srcMatchKey})
	sort.Sort(SortVector{vector1, destMatchKey})

	offset := 0
	leng1 := len(vector1)
out:
	for _, el := range vector {
		if srcTargetKey != "" {
			el.Metric = append(el.Metric, labels.Label{Name: srcNewkey, Value: strconv.FormatFloat(el.V, 'f', -1, 64)})
		}
		srcV := el.Metric.Get(srcMatchKey)
		if srcV == "" { //src无需要匹配
			continue out
		}

		if offset == leng1 { //需要匹配的游标已经到末尾
			break out
		}

		for ; offset < leng1; offset ++ {
			el0 := vector1[offset]
			destV := el0.Metric.Get(destMatchKey)
			if srcV == destV {
				for i, targetKey := range targetKeys {
					if targetKey == TAG_FOR_VALUE+"1" {
						el.Metric = append(el.Metric, labels.Label{Name: newKeys[i], Value: strconv.FormatFloat(el0.V, 'f', -1, 64)})
						continue
					}
					if targetValue := el0.Metric.Get(targetKey); targetValue != "" {
						if index := el.Metric.Index(newKeys[i]); index != -1 {
							el.Metric[index].Value = targetValue
						} else {
							el.Metric = append(el.Metric, labels.Label{Name: newKeys[i], Value: targetValue})
						}
					}
				}
				sort.Sort(el.Metric)
				enh.Out = append(enh.Out, el)
				continue out //这里退出没有进行offset++，因为src可以存在matchKey对应值相同的记录有多条
			}

			//找不到匹配的
			if srcV < destV {
				continue out
			}
		}
	}
	return enh.Out
}

//add by newland
func funcLabelsAppend(vals []parser.Value, args parser.Expressions, enh *EvalNodeHelper) Vector {
	//time.Sleep(time.Millisecond * 500)
	leng := len(args) - 2
	var (
		vector     = vals[0].(Vector)
		vector1    = vals[1].(Vector)
		targetKeys = make([]string, leng/2)
		newKeys    = make([]string, leng/2)
	)

	if enh.Dmn == nil {
		enh.Dmn = make(map[uint64]labels.Labels, len(enh.Out))
	}

	var matchKey string
	for i := 2; i < len(args); i++ {
		src := stringFromArg(args[i])
		if !model.LabelName(src).IsValid() {
			panic(fmt.Errorf("invalid source label name in labels_invert(): %s", src))
		}
		if i == 2 {
			matchKey = src
			continue
		}

		if i%2 == 0 {
			newKeys[(i-2)/2-1] = src
		} else {
			targetKeys[(i-2)/2] = src
		}
	}

	if len(targetKeys) != len(newKeys) {
		panic("invalid params in labels_append()")
	}

	var srcTargetKey, srcNewkey string
	for i, targetKey := range targetKeys {
		if targetKey == TAG_FOR_VALUE+"0" {
			srcTargetKey = targetKey
			srcNewkey = newKeys[i]
		}
	}

	sort.Sort(SortVector{vector, matchKey})
	sort.Sort(SortVector{vector1, matchKey})

	offset := 0
	leng1 := len(vector1)
out:
	for i, el := range vector {
		if srcTargetKey != "" {
			el.Metric = append(el.Metric, labels.Label{Name: srcNewkey, Value: strconv.FormatFloat(el.V, 'f', -1, 64)})
		}
		srcV := el.Metric.Get(matchKey)
		if srcV == "" { //src无需要匹配
			enh.Out = append(enh.Out, el)
			continue out
		}

		if offset == leng1 { //需要匹配的游标已经到末尾
			enh.Out = append(enh.Out, vector[i:]...)
			break out
		}

		for ; offset < leng1; offset ++ {
			el0 := vector1[offset]
			destV := el0.Metric.Get(matchKey)
			if srcV == destV {
				for i, targetKey := range targetKeys {
					if targetKey == TAG_FOR_VALUE+"1" {
						el.Metric = append(el.Metric, labels.Label{Name: newKeys[i], Value: strconv.FormatFloat(el0.V, 'f', -1, 64)})
						continue
					}
					if targetValue := el0.Metric.Get(targetKey); targetValue != "" {
						el.Metric = append(el.Metric, labels.Label{Name: newKeys[i], Value: targetValue})
					}
				}
				enh.Out = append(enh.Out, el)
				continue out //这里退出没有进行offset++，因为src可以存在matchKey对应值相同的记录有多条
			}

			//如果找不到匹配的，也join到结果
			if srcV < destV {
				sort.Sort(el.Metric)
				enh.Out = append(enh.Out, el)
				continue out
			}
		}

		//上面for全部遍历完，未匹配上
		sort.Sort(el.Metric)
		enh.Out = append(enh.Out, el)

		//for _, el0 := range vector1 {
		//	destV := el0.Metric.Get(matchKey)
		//	if srcV == destV {
		//		for i, targetKey := range targetKeys {
		//			if targetKey == TAG_FOR_VALUE+"1" {
		//				el.Metric = append(el.Metric, labels.Label{Name: newKeys[i], Value: strconv.FormatFloat(el0.V, 'f', -1, 64)})
		//				continue
		//			}
		//			if targetValue := el0.Metric.Get(targetKey); targetValue != "" {
		//				el.Metric = append(el.Metric, labels.Label{Name: newKeys[i], Value: targetValue})
		//			}
		//		}
		//		enh.Out = append(enh.Out, el)
		//		continue out
		//	}
		//	//如果找不到匹配的，会剔除el
		//}
	}
	return enh.Out
}

//add by newland
func funcSelect2Join(vals []parser.Value, args parser.Expressions, enh *EvalNodeHelper) Vector {
	leng := len(args) - 3
	var (
		vector     = vals[1].(Vector)
		vector1    = vals[2].(Vector)
		targetKeys = make([]string, leng)
		newKeys    = make([]string, leng)
	)

	srcMatchKey, destMatchKey := strutil.SplitString(stringFromArg(args[0]), SPLIT_SYMBOL_1)
	for i := 3; i < len(args); i++ {
		targetkey, newkey := strutil.SplitString(stringFromArg(args[i]), ":")

		if !model.LabelName(targetkey).IsValid() {
			panic(fmt.Errorf("invalid source label name in select2_join(): %s", targetkey))
		}
		if !model.LabelName(newkey).IsValid() {
			panic(fmt.Errorf("invalid source label name in select2_join(): %s", newkey))
		}
		targetKeys[i-3], newKeys[i-3] = targetkey, newkey
	}

	if enh.Dmn == nil {
		enh.Dmn = make(map[uint64]labels.Labels, len(enh.Out))
	}

	var srcTargetKey, srcNewkey string
	for i, targetKey := range targetKeys {
		if targetKey == TAG_FOR_VALUE+"0" {
			srcTargetKey = targetKey
			srcNewkey = newKeys[i]
		}
	}

	sort.Sort(SortVector{vector, srcMatchKey})
	sort.Sort(SortVector{vector1, destMatchKey})

	offset := 0
	leng1 := len(vector1)
out:
	for i, el := range vector {
		if srcTargetKey != "" {
			el.Metric = append(el.Metric, labels.Label{Name: srcNewkey, Value: strconv.FormatFloat(el.V, 'f', -1, 64)})
		}
		srcV := el.Metric.Get(srcMatchKey)
		if srcV == "" { //src无需要匹配
			enh.Out = append(enh.Out, el)
			continue out
		}

		if offset == leng1 { //需要匹配的游标已经到末尾
			enh.Out = append(enh.Out, vector[i:]...)
			break out
		}

		for ; offset < leng1; offset ++ {
			el0 := vector1[offset]
			destV := el0.Metric.Get(destMatchKey)
			if srcV == destV {
				for i, targetKey := range targetKeys {
					if targetKey == TAG_FOR_VALUE+"1" {
						el.Metric = append(el.Metric, labels.Label{Name: newKeys[i], Value: strconv.FormatFloat(el0.V, 'f', -1, 64)})
						continue
					}
					if targetValue := el0.Metric.Get(targetKey); targetValue != "" {
						el.Metric = append(el.Metric, labels.Label{Name: newKeys[i], Value: targetValue})
					}
				}
				sort.Sort(el.Metric)
				enh.Out = append(enh.Out, el)
				continue out //这里退出没有进行offset++，因为src可以存在matchKey对应值相同的记录有多条
			}

			//如果找不到匹配的，也join到结果
			if srcV < destV {
				sort.Sort(el.Metric)
				enh.Out = append(enh.Out, el)
				continue out
			}
		}

		//上面for全部遍历完，未匹配上
		sort.Sort(el.Metric)
		enh.Out = append(enh.Out, el)

	}
	return enh.Out
}

func funcDistinct(vals []parser.Value, args parser.Expressions, enh *EvalNodeHelper) Vector {
	var (
		vector  = vals[0].(Vector)
		sortKey = stringFromArg(args[1])
	)
	sort.Sort(SortVector{vector, sortKey})

	if len(vector) > 0 {
		newVector := make(Vector, 0, len(vector))
		newVector = append(newVector, vector[0])

		var el *Sample
		offset := 0
		lastSample := &newVector[offset]
		for i := 1; i < len(vector); i++ {
			el = &vector[i]
			if el.Metric.Get(sortKey) == lastSample.Metric.Get(sortKey) {
				if el.T > lastSample.T {
					newVector[offset] = *el
					lastSample = el
				}
			} else {
				newVector = append(newVector, *el)
				offset ++
				lastSample = el
			}
		}

		enh.Out = append(enh.Out, newVector...)
	}

	return enh.Out
}

//add by newland
//这个方法被调用多次，每一个指标调用一次（一个指标有多个监控数据），性能较差
func funcAggregateMatrixMax(vals []parser.Value, args parser.Expressions, enh *EvalNodeHelper) Vector {
	var (
		matrix = vals[0].(Matrix)
	)

	if enh.Dmn == nil {
		enh.Dmn = make(map[uint64]labels.Labels, len(enh.Out))
	}

	for _, el := range matrix {
		var points Points = el.Points
		if lp := len(points); lp > 0 {
			sort.Sort(points)
			s := Sample{
				Metric: el.Metric,
				Point:  points[0],
			}
			sort.Sort(s.Metric)
			enh.Out = append(enh.Out, s)
		}
	}
	return enh.Out
}

//funcAggregateMatrixMin
func funcAggregateMatrixMin(vals []parser.Value, args parser.Expressions, enh *EvalNodeHelper) Vector {
	var (
		matrix = vals[0].(Matrix)
	)

	if enh.Dmn == nil {
		enh.Dmn = make(map[uint64]labels.Labels, len(enh.Out))
	}

	for _, el := range matrix {
		var points Points = el.Points
		if lp := len(points); lp > 0 {
			sort.Sort(points)
			s := Sample{
				Metric: el.Metric,
				Point:  points[lp-1],
			}
			sort.Sort(s.Metric)
			enh.Out = append(enh.Out, s)
		}
	}
	return enh.Out
}

func funcAggregateMatrixSum(vals []parser.Value, args parser.Expressions, enh *EvalNodeHelper) Vector {
	var (
		matrix = vals[0].(Matrix)
	)

	if enh.Dmn == nil {
		enh.Dmn = make(map[uint64]labels.Labels, len(enh.Out))
	}

	for _, el := range matrix {
		var sum float64
		for _, p := range el.Points {
			sum += p.V
		}
		s := Sample{
			Metric: el.Metric,
			Point:  Point{T: el.Points[len(el.Points)-1].T, V: sum},
		}
		sort.Sort(s.Metric)
		enh.Out = append(enh.Out, s)
	}
	return enh.Out
}

//add by newland
func funcLabelsAppendValue(vals []parser.Value, args parser.Expressions, enh *EvalNodeHelper) Vector {
	var (
		vector    = vals[0].(Vector)
		vector1   = vals[1].(Vector)
		srcLabels = make([]string, len(args)-2)
	)

	if enh.Dmn == nil {
		enh.Dmn = make(map[uint64]labels.Labels, len(enh.Out))
	}

	for i := 2; i < len(args); i++ {
		src := stringFromArg(args[i])
		if !model.LabelName(src).IsValid() {
			panic(fmt.Errorf("invalid source label name in labels_invert(): %s", src))
		}
		srcLabels[i-2] = src
	}

	matchKey := srcLabels[0]
	newKey := srcLabels[1]

out:
	for _, el := range vector {
		srcV := el.Metric.Get(matchKey)
		if srcV == "" {
			enh.Out = append(enh.Out, el)
			continue out
		}

		for _, el0 := range vector1 {
			destV := el0.Metric.Get(matchKey)
			if srcV == destV {
				el.Metric = append(el.Metric, labels.Label{Name: newKey, Value: strconv.FormatFloat(el0.V, 'f', -1, 64)})
				sort.Sort(el.Metric)
				enh.Out = append(enh.Out, el)
				continue out
			}
		}

	}
	return enh.Out
}

//funcLabelsAppendValue2  ("pod_name", "cpu|mem", app_cpu_used, app_mem_used)
func funcLabelsAppendValue2(vals []parser.Value, args parser.Expressions, enh *EvalNodeHelper) Vector {
	matchKeys := strings.Split(stringFromArg(args[0]), ",")
	newKeys := strings.Split(stringFromArg(args[1]), "|")

	vs := make([]Vector, len(args)-2)
	for i, leng := 2, len(args); i < leng; i++ {
		vs[i-2] = vals[i].(Vector)
	}

	if len(matchKeys) == 0 {
		panic(fmt.Errorf("empty matchKey in labels_append_values()"))
	}

	for i, matchKey := range matchKeys {
		if !model.LabelName(matchKey).IsValid() {
			panic(fmt.Errorf("invalid source label name in labels_append_value2(): %s", matchKey))
		}
		matchKeys[i] = strings.TrimSpace(matchKey)
	}

	if len(newKeys) != len(vs) {
		panic(fmt.Errorf("invalid target label name in labels_append_value2(): %s", args[1].(*parser.StringLiteral).Val))
	}

	if enh.Dmn == nil {
		enh.Dmn = make(map[uint64]labels.Labels, len(enh.Out))
	}

	var srcVs = make([]string, len(matchKeys))
	var matched = true

out:
	for _, e0 := range vs[0] {
		for i, matchKey := range matchKeys {
			srcV := e0.Metric.Get(matchKey)
			if srcV == "" {
				continue out
			}
			srcVs[i] = srcV
		}

		e0.Metric = append(e0.Metric, labels.Label{Name: newKeys[0], Value: strconv.FormatFloat(e0.V, 'f', -1, 64)})

		for i, leng := 1, len(vs); i < leng; i++ {
			for _, el0 := range vs[i] {
				matched = true
				for i, matchKey := range matchKeys {
					if srcVs[i] != el0.Metric.Get(matchKey) {
						matched = false
						break
					}
				}

				if matched {
					e0.Metric = append(e0.Metric, labels.Label{Name: newKeys[i], Value: strconv.FormatFloat(el0.V, 'f', -1, 64)})
					break
				}
			}
		}
		sort.Sort(e0.Metric)
		enh.Out = append(enh.Out, e0)

	}

	return enh.Out
}

func funcSelectn(vals []parser.Value, args parser.Expressions, enh *EvalNodeHelper) Vector {
	matchKeys := strings.Split(stringFromArg(args[0]), SPLIT_SYMBOL_1)
	newKeys := strings.Split(stringFromArg(args[1]), SPLIT_SYMBOL_1)

	vs := make([]Vector, len(args)-2)
	for i, leng := 2, len(args); i < leng; i++ {
		vs[i-2] = vals[i].(Vector)
	}

	for i, matchKey := range matchKeys {
		if !model.LabelName(matchKey).IsValid() {
			panic(fmt.Errorf("invalid source label name in selectn(): %s", matchKey))
		}
		matchKeys[i] = strings.TrimSpace(matchKey)
	}
	lengmatchKeys := len(matchKeys)
	if lengmatchKeys == 0 || !(lengmatchKeys == len(vs) || lengmatchKeys == 1) {
		panic(fmt.Errorf("invalid matchKey in selectn()"))
	}

	if len(newKeys) != len(vs) {
		panic(fmt.Errorf("invalid target label name in selectn(): %s", args[1].(*parser.StringLiteral).Val))
	}

	if enh.Dmn == nil {
		enh.Dmn = make(map[uint64]labels.Labels, len(enh.Out))
	}

	var srcMatchValue string

out:
	for _, e0 := range vs[0] {
		if srcMatchValue = e0.Metric.Get(matchKeys[0]); srcMatchValue == "" {
			continue out
		}

		e0.Metric = append(e0.Metric, labels.Label{Name: newKeys[0], Value: strconv.FormatFloat(e0.V, 'f', -1, 64)})
		for i, leng := 1, len(vs); i < leng; i++ {
			for _, el0 := range vs[i] {
				var destmatchKey string
				if lengmatchKeys == 1 {
					destmatchKey = matchKeys[0]
				} else {
					destmatchKey = matchKeys[i]
				}
				if srcMatchValue == el0.Metric.Get(destmatchKey) {
					e0.Metric = append(e0.Metric, labels.Label{Name: newKeys[i], Value: strconv.FormatFloat(el0.V, 'f', -1, 64)})
					break
				}
			}
		}
		sort.Sort(e0.Metric)
		enh.Out = append(enh.Out, e0)

	}

	return enh.Out
}

//add by newland 通过label 反选指标
func funcLabelsInvert(vals []parser.Value, args parser.Expressions, enh *EvalNodeHelper) Vector {
	var (
		vector    = vals[0].(Vector)
		srcLabels = make([]string, len(args)-1)
	)

	if enh.Dmn == nil {
		enh.Dmn = make(map[uint64]labels.Labels, len(enh.Out))
	}

	for i := 1; i < len(args); i++ {
		src := stringFromArg(args[i])
		if !model.LabelName(src).IsValid() {
			panic(fmt.Errorf("invalid source label name in labels_invert(): %s", src))
		}
		srcLabels[i-1] = src
	}

out:
	for _, el := range vector {

		for _, src := range srcLabels {
			if el.Metric.Get(src) != "" {
				continue out
			}
		}
		sort.Sort(el.Metric)
		enh.Out = append(enh.Out, el)

	}
	return enh.Out
}

// === Vector(s Scalar) Vector === //newland
func funcVectorWithLabels(vals []parser.Value, args parser.Expressions, enh *EvalNodeHelper) Vector {

	var (
		v = vals[0].(Vector)[0].V
	)

	metriclabels := make(labels.Labels, 0, len(args)/2)
	var isKey = true
	var l *labels.Label
	for i := 1; i < len(args); i++ {
		src := stringFromArg(args[i])
		if isKey {
			if !model.LabelName(src).IsValid() {
				panic(fmt.Errorf("invalid source label name in vector_with_labels(): %s", src))
			}
			l = &labels.Label{}
			l.Name = src
			isKey = false
		} else {
			l.Value = src
			metriclabels = append(metriclabels, *l)
			isKey = true
		}

	}

	return append(enh.Out,
		Sample{
			Metric: metriclabels,
			Point:  Point{V: v},
		})
}

// funcVectorJoin
func funcVectorJoin(vals []parser.Value, args parser.Expressions, enh *EvalNodeHelper) Vector {
	for _, v := range vals {
		enh.Out = append(enh.Out, v.(Vector)...)
	}
	return enh.Out
}
