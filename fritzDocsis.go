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
			Docsis31 []struct {
				PowerLevel string `json:"powerLevel"`
				Type       string `json:"type"`
				Channel    int    `json:"channel"`
				ChannelID  int    `json:"channelID"`
				Frequency  string `json:"frequency"`
			} `json:"docsis31"`
			Docsis30 []struct {
				Type          string  `json:"type"`
				CorrErrors    float64 `json:"corrErrors"`
				Mse           string  `json:"mse"`
				PowerLevel    string  `json:"powerLevel"`
				Channel       int     `json:"channel"`
				NonCorrErrors float64 `json:"nonCorrErrors"`
				Latency       float64 `json:"latency"`
				ChannelID     int     `json:"channelID"`
				Frequency     string  `json:"frequency"`
			} `json:"docsis30"`
		} `json:"channelDs"`
		Oem       string `json:"oem"`
		ChannelUs struct {
			Docsis30 []struct {
				PowerLevel string `json:"powerLevel"`
				Type       string `json:"type"`
				Channel    int    `json:"channel"`
				Multiplex  string `json:"multiplex"`
				ChannelID  int    `json:"channelID"`
				Frequency  string `json:"frequency"`
			} `json:"docsis30"`
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

func setMetrics(data *docInfo) {
	for _, channel := range data.Data.ChannelDs.Docsis30 {
		labels := prometheus.Labels{
			"channel":       strconv.Itoa(channel.Channel),
			"channelID":     strconv.Itoa(channel.ChannelID),
			"direction":     "downstream",
			"frequency":     channel.Frequency,
			"docsisVersion": "3.0",
		}
		correctableErrors.With(labels).Set(float64(channel.CorrErrors))
		uncorrectableErrors.With(labels).Set(float64(channel.NonCorrErrors))
		mseData, _ := strconv.ParseFloat(channel.Mse, 64)
		powerLevelData, _ := strconv.ParseFloat(channel.PowerLevel, 64)
		mse.With(labels).Set(mseData)
		powerLevel.With(labels).Set(powerLevelData)
		connectionTypeData, _ := strconv.ParseFloat(strings.TrimSuffix(channel.Type, "QAM"), 64)
		connectionType.With(labels).Set(connectionTypeData)
	}
	for _, channel := range data.Data.ChannelDs.Docsis31 {
		labels := prometheus.Labels{
			"channel":       strconv.Itoa(channel.Channel),
			"channelID":     strconv.Itoa(channel.ChannelID),
			"direction":     "downstream",
			"frequency":     channel.Frequency,
			"docsisVersion": "3.1",
		}
		powerLevelData, _ := strconv.ParseFloat(channel.PowerLevel, 64)
		powerLevel.With(labels).Set(powerLevelData)
		connectionTypeData, _ := strconv.ParseFloat(strings.TrimSuffix(channel.Type, "QAM"), 64)
		connectionType.With(labels).Set(connectionTypeData)
	}
	for _, channel := range data.Data.ChannelUs.Docsis30 {
		labels := prometheus.Labels{
			"channel":       strconv.Itoa(channel.Channel),
			"channelID":     strconv.Itoa(channel.ChannelID),
			"direction":     "upstream",
			"frequency":     channel.Frequency,
			"docsisVersion": "3.0",
		}
		powerLevelData, _ := strconv.ParseFloat(channel.PowerLevel, 64)
		powerLevel.With(labels).Set(powerLevelData)
		connectionTypeData, _ := strconv.ParseFloat(strings.TrimSuffix(channel.Type, "QAM"), 64)
		connectionType.With(labels).Set(connectionTypeData)
	}
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
