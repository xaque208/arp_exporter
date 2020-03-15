package exporter

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	junos "github.com/scottdware/go-junos"
	log "github.com/sirupsen/logrus"
	"github.com/xaque208/znet/znet"
)

var (
	// macAddress = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	// 	Name: "mac_known",
	// 	Help: "Known mac addresses",
	// }, []string{"ip", "mac"})

	scrapeDurationMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "arp_scrape_duration",
		Help: "The duration in seconds for an individual scrape",
	}, []string{"hostname"})

	goodScrapeCountMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "arp_good_scrape_count",
		Help: "The duration in seconds for an individual scrape",
	})

	knownHostsMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "arp_known_hosts",
	})

	unknownHostsMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "arp_unknown_hosts",
	})

	arpEntriesMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "arp_entries",
		Help: "Entries from the ARP table",
	}, []string{"state"})
)

func init() {
	prometheus.MustRegister(
		goodScrapeCountMetric,
		knownHostsMetric,
		scrapeDurationMetric,
		unknownHostsMetric,
		arpEntriesMetric,
	)
}

func hasMAC(mac string, hosts []znet.NetworkHost) bool {

	for _, h := range hosts {
		for _, m := range h.MACAddress {
			if noramlMAC(mac) == noramlMAC(m) {
				return true
			}
		}
	}

	return false
}

func watchMAC(mac string, hosts []znet.NetworkHost) bool {

	for _, h := range hosts {
		if h.Watch {
			return true
		}
	}

	return false
}

func noramlMAC(mac string) string {
	m := strings.ToLower(mac)
	return m
}

// ScrapeMetrics performs the necessary calls to retreive the current arp tables from each device.
func ScrapeMetrics(auth *junos.AuthMethod, hosts []znet.NetworkHost, unknownChan chan junos.ArpEntry) {
	goodScrape := 0
	knownHosts := 0
	unknownHosts := 0
	watchHosts := 0

	for _, h := range hosts {

		if h.Platform == "" {
			continue
		}

		if h.HostName == "" {
			continue
		}

		if h.Platform == "junos" {
			now := time.Now()

			// go func() {
			log.Debugf("Scraping host: %s", h.HostName)
			session, err := junos.NewSession(h.HostName, auth)
			if err != nil {
				log.Error(err)
				continue
			}
			defer session.Close()

			views, err := session.View("arp")
			if err != nil {
				log.Error(err)
				continue
			}

			for _, arp := range views.Arp.Entries {

				if arp.Interface == "ppp0.0" {
					continue
				}

				// macAddress.WithLabelValues(arp.MACAddress, arp.IPAddress).Set(1)
				if hasMAC(arp.MACAddress, hosts) {
					knownHosts++
				} else {
					unknownHosts++
					unknownChan <- arp
				}

				if watchMAC(arp.MACAddress, hosts) {
					watchHosts++
				}
			}

			goodScrape++

			seconds := time.Since(now).Seconds()
			scrapeDurationMetric.WithLabelValues(h.HostName).Set(seconds)

			// }()
		}
	}

	log.Debugf("KnownHosts %d", knownHosts)
	log.Debugf("UnknownHosts %d", unknownHosts)
	log.Debugf("WatchHosts %d", watchHosts)

	goodScrapeCountMetric.Set(float64(goodScrape))
	arpEntriesMetric.WithLabelValues("known").Set(float64(knownHosts))
	arpEntriesMetric.WithLabelValues("unknown").Set(float64(unknownHosts))

}
