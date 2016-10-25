package main

import (
	"expvar"
	"strconv"
	"sync/atomic"
	"time"
)

var (
	parseFormErrors    *expvar.Int
	parseRequestErrors *expvar.Int
	proxyErrors        *expvar.Int
	copyErrors         *expvar.Int

	numRequests     *expvar.Int
	proxiedRequests *expvar.Int

	avgUpstreamTime *expvar.Float
	avgTotalTime    *expvar.Float

	upstreamRequestTime int64
	totalRequestTime    int64
)

func initCounters() {
	parseFormErrors = expvar.NewInt("parseFormErrors")
	parseRequestErrors = expvar.NewInt("parseRequestErrors")
	proxyErrors = expvar.NewInt("proxyErrors")
	copyErrors = expvar.NewInt("copyErrors")

	numRequests = expvar.NewInt("numRequests")
	proxiedRequests = expvar.NewInt("proxiedRequests")

	avgUpstreamTime = expvar.NewFloat("avgUpstreamTime")
	avgTotalTime = expvar.NewFloat("avgTotalTime")

	upstreamRequestTime = 0
	totalRequestTime = 0
}

func milliseconds(t time.Duration) int64 {
	ns := t.Nanoseconds()
	ms := ns / (int64(time.Millisecond) / int64(time.Nanosecond))
	return ms
}

func updateCounters(total, proxy time.Duration) {
	proxy_time := atomic.AddInt64(&upstreamRequestTime, milliseconds(proxy))
	total_time := atomic.AddInt64(&totalRequestTime, milliseconds(total))

	req, err := strconv.ParseFloat(proxiedRequests.String(), 64)
	if err != nil {
		return
	}

	avgUpstreamTime.Set(float64(proxy_time) / req)
	avgTotalTime.Set(float64(total_time) / req)
}
