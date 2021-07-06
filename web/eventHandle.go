package web


import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"sync"
)

var eventCollector *EventCollector
var once2 sync.Once

func GetEventCollector() *EventCollector {
	once2.Do(func() {
		eventCollector = NewEventCollector()
	})

	return eventCollector
}

type PodEvent struct {
	Message    string `json:"message"`
	Host       string `json:"host"`
	Namespace  string `json:"namespace"`
	PodName    string `json:"pod_name"`
	OprType    string `json:"opr_type"`
	EventCount int    `json:"event_count"`
}

func NewHttpHandler() http.Handler {
	workerDB2 := GetEventCollector()

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(workerDB2)

	gatherers := prometheus.Gatherers{
		prometheus.DefaultGatherer,
		reg,
	}

	h := promhttp.HandlerFor(gatherers,
		promhttp.HandlerOpts{
			ErrorLog:      &Mylogger{},
			ErrorHandling: promhttp.ContinueOnError,
		})

	return h
}

type EventCollector struct {
	EventDesc *prometheus.Desc
	events    chan *PodEvent
	lock      sync.Mutex
}

func NewEventCollector() *EventCollector {
	return &EventCollector{
		EventDesc: prometheus.NewDesc(
			"app_pod_event",
			"app pod event.",
			[]string{"message", "host", "namespace", "pod_name", "opr_type"},
			prometheus.Labels{},
		),
		events: make(chan *PodEvent, 10000),
	}
}

func (this *EventCollector) Push(e *PodEvent) {
	//this.lock.Lock()
	//defer this.lock.Unlock()
	this.events <- e
}

func (this *EventCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- this.EventDesc
}

func (this *EventCollector) Collect(ch chan<- prometheus.Metric) {
	//this.lock.Lock()
	//defer this.lock.Unlock()
	count := 0
	for {
		if count > 9999 {
			return
		}
		select {
		case e := <-this.events:
			ch <- prometheus.MustNewConstMetric(
				this.EventDesc,
				prometheus.CounterValue,
				float64(e.EventCount),
				e.Message, e.Host, e.Namespace, e.PodName, e.OprType, //可变label值
			)
			count++
		default:
			return
		}

	}
}

type Mylogger struct {
}

func (l *Mylogger) Println(v ...interface{}) {
	fmt.Println(v)
}
