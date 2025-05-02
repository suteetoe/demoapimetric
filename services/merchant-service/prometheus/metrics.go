package prometheus

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var CreateMerchantCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "merchant_create_total",
		Help: "Total number of merchant creations",
	},
)

var GetMerchantCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "merchant_get_total",
		Help: "Total number of merchant retrievals",
	},
)

var ListMerchantsCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "merchant_list_total",
		Help: "Total number of merchant listing requests",
	},
)

func init() {
	prometheus.MustRegister(CreateMerchantCounter)
	prometheus.MustRegister(GetMerchantCounter)
	prometheus.MustRegister(ListMerchantsCounter)
}

func GetPrometheusHandler() http.Handler {
	return promhttp.Handler()
}
