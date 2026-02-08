package metric

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"time"
)

type MiddlewareBuilder struct {
	Namespace  string
	Subsystem  string
	Name       string
	Help       string // 提示信息
	InstanceID string
}

func (m *MiddlewareBuilder) Build() gin.HandlerFunc {
	// pattern 是指你命中的路由
	// 指 http 的 status
	labels := []string{"method", " ", "status"}
	summary := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: m.Namespace,
		Subsystem: m.Subsystem,
		Name:      m.Name + "_resp_time",
		Help:      m.Help,
		ConstLabels: map[string]string{
			"instance_id": m.InstanceID,
		},
		Objectives: map[float64]float64{
			0.5:   0.01,
			0.9:   0.01,
			0.99:  0.005,
			0.999: 0.0001,
		},
	}, labels)
	// 这里如果 panic 说明上面的 Namespace+Subsystem+Name 冲突了
	prometheus.MustRegister(summary)

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: m.Namespace,
		Subsystem: m.Subsystem,
		Name:      m.Name + "_active_time",
		Help:      m.Help,
		ConstLabels: map[string]string{
			"instance_id": m.InstanceID,
		},
	})
	prometheus.MustRegister(gauge)

	return func(ctx *gin.Context) {
		start := time.Now()
		gauge.Inc()
		// 防止 panic 所以放到 defer 中
		defer func() {
			duration := time.Since(start)
			gauge.Dec()
			pattern := ctx.FullPath() // 命中路由
			// 路由可能 404
			if pattern == "" {
				pattern = "unknown"
			}
			// WithLabelValues:参数个数对应 labels 个数
			summary.WithLabelValues(
				ctx.Request.Method,
				pattern,
				strconv.Itoa(ctx.Writer.Status()),
				// Observe: 统计执行时间
			).Observe(float64(duration.Milliseconds()))
		}()

		// 最终执行到业务里
		ctx.Next()

	}
}
