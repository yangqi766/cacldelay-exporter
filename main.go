package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	cmd           *exec.Cmd
	stdout        bytes.Buffer
	stderr        bytes.Buffer
	timeLayoutStr string
	histogram     = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "inno",
			Name:      "my",
			Help:      "histogram",
		})
)

func tailFile(rows int, path string) bytes.Buffer {
	cmdstr := fmt.Sprintf("tail -%d %s", rows, path)
	fmt.Println(cmdstr)
	cmd = exec.Command("bash", "-c", cmdstr)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return stdout
}

func toTimestamp(timestr string) int64 {
	hour, _ := strconv.ParseInt(timestr[:2], 10, 64)
	minute, _ := strconv.ParseInt(timestr[2:4], 10, 64)
	second, _ := strconv.ParseInt(timestr[4:6], 10, 64)
	nanosecond, _ := strconv.ParseInt(timestr[6:9], 10, 64)
	timestamp := (hour*3600+minute*60+second)*1000 + nanosecond
	return timestamp
}

func delayList() map[string][]string {
	delaymap := map[string][]string{}
	output := tailFile(10, "/home/xiaoma/cacldelay-exporter/Trade.csv-20210528")
	newout := strings.Split(output.String(), "\n")
	for _, line := range newout {
		if line == "" {
			continue
		}
		array := strings.Split(line, ",")
		stonecode := array[0]

		jpstime := toTimestamp(array[2])
		recvtime := toTimestamp(array[16][8:17])

		delay := fmt.Sprintf("%d", recvtime-jpstime)

		_, ok := delaymap[stonecode]
		if !ok {
			fmt.Println(1111111111111)
			delaymap[stonecode] = []string{delay}
		} else {
			delaymap[stonecode] = append(delaymap[stonecode], delay)
		}

		fmt.Println(delay)
	}
	fmt.Println(delaymap)
	return delaymap
}

func main() {
	histogramVec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "prom_request_time",
		Help: "Time it has taken to retrieve the metrics",
	}, []string{})

	prometheus.Register(histogramVec)

	http.Handle("/metrics", newHandlerWithHistogram(promhttp.Handler(), histogramVec))

	prometheus.MustRegister(histogram)
}

func newHandlerWithHistogram(handler http.Handler, histogram *prometheus.HistogramVec) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		status := http.StatusOK

		delaymap := delayList()

		for k, v := range delaymap {
			for vv := range v {
				histogram.WithLabelValues(fmt.Sprintf("%s", k)).Observe(vv)
			}
		}

		if req.Method == http.MethodGet {
			handler.ServeHTTP(w, req)
			return
		}
		status = http.StatusBadRequest

		w.WriteHeader(status)
	})
}
