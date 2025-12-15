package ioc

import (
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

func InitPrometheus() *prometheus.Registry {
	// 创建一个 registry
	reg := prometheus.NewRegistry()

	// 添加 Go 编译信息
	reg.MustRegister(collectors.NewBuildInfoCollector())
	// Go runtime metrics
	reg.MustRegister(collectors.NewGoCollector(
		collectors.WithGoCollectorRuntimeMetrics(
			collectors.GoRuntimeMetricsRule{Matcher: regexp.MustCompile("/.*")},
		),
	))

	return reg
}
