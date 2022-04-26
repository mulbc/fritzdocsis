package main

import (
	"flag"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/philippfranke/go-fritzbox/fritzbox"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

type channelInfo []struct {
	Activesub     string  `json:"activesub"`
	Channel       int     `json:"channel"`
	ChannelID     int     `json:"channelID"`
	CorrErrors    float64 `json:"corrErrors"`
	FFT           string  `json:"fft"`
	Frequency     string  `json:"frequency"`
	Latency       float64 `json:"latency"`
	Mer           string  `json:"mer"`
	Modulation    string  `json:"modulation"`
	Mse           string  `json:"mse"`
	Multiplex     string  `json:"multiplex"`
	NonCorrErrors float64 `json:"nonCorrErrors"`
	PLC           string  `json:"plc"`
	PowerLevel    string  `json:"powerLevel"`
	Type          string  `json:"type"`
}

type docInfo struct {
	Pid  string `json:"pid"`
	Hide struct {
		Wps        bool `json:"wps"`
		ShareUsb   bool `json:"shareUsb"`
		LiveTv     bool `json:"liveTv"`
		FaxSet     bool `json:"faxSet"`
		WGuest     bool `json:"wGuest"`
		DectMoniEx bool `json:"dectMoniEx"`
		Rss        bool `json:"rss"`
		Mobile     bool `json:"mobile"`
		WKey       bool `json:"wKey"`
		SsoSet     bool `json:"ssoSet"`
		DectMail   bool `json:"dectMail"`
		DectMoni   bool `json:"dectMoni"`
		Chan       bool `json:"chan"`
		LiveImg    bool `json:"liveImg"`
	} `json:"hide"`
	Time []interface{} `json:"time"`
	Data struct {
		ChannelDs struct {
			Docsis31 channelInfo `json:"docsis31"`
			Docsis30 channelInfo `json:"docsis30"`
		} `json:"channelDs"`
		Oem       string `json:"oem"`
		ChannelUs struct {
			Docsis31 channelInfo `json:"docsis31"`
			Docsis30 channelInfo `json:"docsis30"`
		} `json:"channelUs"`
	} `json:"data"`
	Sid string `json:"sid"`
}

func startPrometheusExporter() {
	http.Handle("/metrics", promhttp.Handler())
	log.Info("Starting Exporter on Port 2112")
	log.WithError(http.ListenAndServe(":2112", nil)).Fatal("Issues with serving exporter on port")
}

func collectFritzMetrics(client *fritzbox.Client) (docInfo, error) {
	requestValues := url.Values{}
	requestValues.Add("page", "docInfo")
	request, err := client.NewRequest("POST", "data.lua", requestValues)
	if err != nil {
		log.WithError(err).Fatal("Issues creating fritzbox request")
	}
	var data docInfo
	_, err = client.Do(request, &data)
	if err != nil {
		log.WithError(err).Fatal("Issues getting fritzbox response")
	}
	return data, nil
}

var (
	data              docInfo
	correctableErrors = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "fritz_channel_correctable_errors",
			Help: "The total number of correctable errors on the channel",
		},
		[]string{"channel", "channelID", "direction", "frequency", "docsisVersion"})
	uncorrectableErrors = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "fritz_channel_uncorrectable_errors",
			Help: "The total number of uncorrectable errors on the channel",
		},
		[]string{"channel", "channelID", "direction", "frequency", "docsisVersion"})
	mse = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "fritz_channel_mse",
			Help: "The current Mean Squared Error of the channel",
		},
		[]string{"channel", "channelID", "direction", "frequency", "docsisVersion"})
	mer = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "fritz_channel_mer",
			Help: "The current modulation error ratio of the channel",
		},
		[]string{"channel", "channelID", "direction", "frequency", "docsisVersion"})
	powerLevel = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "fritz_channel_power_level",
			Help: "The current power level of the channel",
		},
		[]string{"channel", "channelID", "direction", "frequency", "docsisVersion"})
	connectionType = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "fritz_channel_connection_type",
			Help: "The current QAM type of the channel",
		},
		[]string{"channel", "channelID", "direction", "frequency", "docsisVersion"})
)

func exportChannelInfo(channels channelInfo, direction string, docsisVersion string) {
	for _, channel := range channels {
		var channel_label int
		// keep channel and channelid same on 7.39
		if channel.Channel == 0 {
			channel_label = channel.ChannelID
		} else {
			channel_label = channel.Channel
		}
		labels := prometheus.Labels{
			"channel":       strconv.Itoa(channel_label),
			"channelID":     strconv.Itoa(channel.ChannelID),
			"direction":     direction,
			"frequency":     channel.Frequency,
			"docsisVersion": docsisVersion,
		}
		correctableErrors.With(labels).Set(float64(channel.CorrErrors))
		uncorrectableErrors.With(labels).Set(float64(channel.NonCorrErrors))
		mseData, _ := strconv.ParseFloat(channel.Mse, 64)
		mse.With(labels).Set(mseData)
		merData, _ := strconv.ParseFloat(channel.Mer, 64)
		mer.With(labels).Set(merData)
		powerLevelData, _ := strconv.ParseFloat(channel.PowerLevel, 64)
		powerLevel.With(labels).Set(powerLevelData)
		var connectionTypeData float64
		if channel.Type == "" {
			connectionTypeData, _ = strconv.ParseFloat(strings.TrimSuffix(channel.Modulation, "QAM"), 64)
		} else {
			connectionTypeData, _ = strconv.ParseFloat(strings.TrimSuffix(channel.Type, "QAM"), 64)
		}
		connectionType.With(labels).Set(connectionTypeData)
	}
}

func setMetrics(data *docInfo) {
	exportChannelInfo(data.Data.ChannelDs.Docsis30, "downstream", "3.0")
	exportChannelInfo(data.Data.ChannelDs.Docsis31, "downstream", "3.1")
	exportChannelInfo(data.Data.ChannelUs.Docsis30, "upstream", "3.0")
	exportChannelInfo(data.Data.ChannelUs.Docsis31, "upstream", "3.1")
}

func main() {
	flagFritzURL := flag.String("url", "http://192.168.178.1", "URL of Fritzbox")
	flagUsername := flag.String("username", "", "Username used for FritzBox authentication [Required]")
	flagPassword := flag.String("password", "", "Password used for FritzBox authentication [Required]")
	flag.Parse()

	if *flagUsername == "" || *flagPassword == "" {
		log.Fatal("Username and Password need to be supplied")
	}

	client := fritzbox.NewClient(nil)
	client.BaseURL, _ = url.Parse(*flagFritzURL)
	err := client.Auth(*flagUsername, *flagPassword)
	if err != nil {
		log.WithError(err).Fatal("Could not log in")
	}
	go func() {
		for {
			data, err = collectFritzMetrics(client)
			setMetrics(&data)
			time.Sleep(2 * time.Second)
		}
	}()

	startPrometheusExporter()
}
