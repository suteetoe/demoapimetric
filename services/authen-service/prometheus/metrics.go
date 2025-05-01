package prometheus

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var LoginCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "auth_login_total",
		Help: "Total number of logins",
	},
)

var RegisterCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "auth_register_total",
		Help: "Total number of user registrations",
	},
)

func init() {
	prometheus.MustRegister(LoginCounter)
	prometheus.MustRegister(RegisterCounter)
}

func GetPrometheusHandler() http.Handler {
	return promhttp.Handler()
}
