package main

import (
	log "github.com/cihub/seelog"
	"github.com/prometheus/client_golang/prometheus"
	g "github.com/soniah/gosnmp"
	"sync"
	"time"
)

const (
	namespace = "snmp"
)

var (
	msgDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "oid", "message"),
		"snmp scrape get desc from oid",
		[]string{"host", "oid", "metric", "msg"},
		nil,
	)

	numberDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "oid", "number"),
		"snmp scrape get number value from oid ",
		[]string{"host", "oid", "metric"},
		nil,
	)
	upDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"'1' if a  scrape of the SNMP device was successful, '0' otherwise.",
		[]string{"host"},
		nil,
	)
	errDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "oid", "err"),
		"snmp scrape failed msg",
		[]string{"host", "oid","metric", "msg"},
		nil,
	)
	durationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape_duration", "seconds"),
		"return how long the scrape took to complete in seconds.",
		[]string{"host"},
		nil,
	)
)

type collector struct{}

func (c collector) Describe(descs chan<- *prometheus.Desc) {
	descs <- upDesc
	descs <- durationDesc
}

func (c collector) Collect(c2 chan<- prometheus.Metric) {
	lock.RLock()
	for _, metric := range metrics {
		if metric != nil {
			c2 <- metric
		}
	}
	lock.RUnlock()
}
func SnmpCollect(target Target) []prometheus.Metric {
	var up int
	var snmpMetrics []prometheus.Metric
	var getMetrics []prometheus.Metric
	start := time.Now()

	up, getMetrics = getSnmp(target)
	snmpMetrics = append(snmpMetrics, getMetrics...)

	snmpMetrics = append(snmpMetrics, prometheus.MustNewConstMetric(
		upDesc,
		prometheus.GaugeValue,
		float64(up),
		target.Host,
	))

	duration := time.Since(start).Seconds()
	durationMetrics := prometheus.MustNewConstMetric(
		durationDesc,
		prometheus.GaugeValue,
		duration,
		target.Host,
	)
	snmpMetrics = append(snmpMetrics, durationMetrics)
	return snmpMetrics
}

func getSnmp(target Target) (int, []prometheus.Metric) {
	var scrapeMetrics []prometheus.Metric
	var errMetrics []prometheus.Metric
	snmp := &g.GoSNMP{
		Target:    target.Host,
		Port:      target.Port,
		Community: "public",
		Version:   g.Version1,
		Timeout:   time.Duration(config.Global.Oidtimeout) * time.Second,
		Retries: config.Global.Oidretries,
	}
	err := snmp.Connect()
	if err != nil {
		_ = log.Errorf("Connect() "+target.Host+" err: %v", err)
		scrapeMetrics = append(scrapeMetrics, prometheus.MustNewConstMetric(
			errDesc,
			prometheus.GaugeValue,
			0,
			target.Host,
			"",
			err.Error(),
		))
		return 0, nil
	}
	defer snmp.Conn.Close()

	//oidMap := make(map[string]string)
	//for _, oid := range target.Oids {
	//	oidMap[oid.Oid] = oid.Metricname
	//}
	wg := sync.WaitGroup{}
	wg.Add(len(target.Oids))
	for i := 0; i < len(target.Oids); i++ {
		go func(i int) {
			result, err := snmp.Get([]string{target.Oids[i].Oid})
			if err != nil {
				_ = log.Errorf("Get() "+ target.Host + target.Oids[i].Metricname + " err: %v", err)
				errMetrics = append(errMetrics, prometheus.MustNewConstMetric(
					errDesc,
					prometheus.GaugeValue,
					0,
					target.Host,
					target.Oids[i].Oid,
					target.Oids[i].Metricname,
					err.Error(),
				))
			} else {
				for _, v := range result.Variables {
					switch v.Type {
					case g.OctetString:
						scrapeMetrics = append(scrapeMetrics, prometheus.MustNewConstMetric(
							msgDesc,
							prometheus.GaugeValue,
							0,
							target.Host,
							target.Oids[i].Oid,
							target.Oids[i].Metricname,
							v.Value.(string),
						))
						//fmt.Printf("string: %s\n", string(v.Value.(string)))
					default:
						scrapeMetrics = append(scrapeMetrics, prometheus.MustNewConstMetric(
							numberDesc,
							prometheus.GaugeValue,
							float64(g.ToBigInt(v.Value).Uint64()),
							target.Host,
							target.Oids[i].Oid,
							target.Oids[i].Metricname,
						))
						//fmt.Printf("number: %d\n", g.ToBigInt(v.Value))
					}
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	if len(errMetrics) == len(target.Oids) {
		return 0, errMetrics
	} else {
		return 1, append(scrapeMetrics, errMetrics...)
	}
}
