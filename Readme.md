# FritzBox Docsis Monitoring

This project contains the fritzdocsis source code that can be used to monitor the coax downstream connection quality.
This project is an [Exporter for Prometheus](https://prometheus.io/docs/instrumenting/exporters/). To use it, you will need a [Prometheus server](https://prometheus.io/) that scrapes this exporter and [Grafana](https://grafana.com/) to display the result of the scrape.

## Building and Running the Exporter

To run the exporter:

1. Clone this repository and cd into it
1. Run `go mod tidy` to get the dependencies
1. Run `go install fritzDocsis.go` to build the binary for your platform
1. Run `${GOPATH}/bin/fritzDocsis` to run the binary

To make live easier, this repository contains a systemd service that you can use to run and auto-start the exporter once you built and installed it.

## Scraping the Exporter via Prometheus

How to install and set up Promtheus is out of scope, but I recommend running Prometheus as a Docker container.

This exporter will be available on Port 2112.

## Looking at the data from Grafana

I have created a very simple Grafana dashboard that will show the error rate and Mean Squared Error (MSE) as well as the channel power level over time.
[The dashboard is available here](https://github.com/mulbc/fritzdocsis/raw/master/doc-assets/grafana-dashboard.json)

Once deployed, the dashboard should look similar to this:

![Grafana Dashboard](https://github.com/mulbc/fritzdocsis/raw/master/doc-assets/grafana-dashboard.jpg "Grafana Dashboard")
