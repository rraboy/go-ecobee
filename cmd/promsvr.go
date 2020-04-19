// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"log"
	"net/http"
	"strconv"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron"
	"github.com/rspier/go-ecobee/ecobee"
	"github.com/spf13/cobra"
)

var listenAddr string

// promCmd represents the status command
var promSvrCmd = &cobra.Command{
	Use:   "promsvr",
	Short: "prometheus metrics server.",
	Long:  "prometheus metrics server.",
	Run: func(cmd *cobra.Command, args []string) {
		checkRequiredFlags()

		if listenAddr == "" {
			glog.Exit("required flag --listen missing")
		}

		promListen()
	},
}

func init() {
	RootCmd.AddCommand(promSvrCmd)
	promSvrCmd.Flags().StringVarP(&listenAddr, "listen", "", ":9442", "the listening address")
	promSvrCmd.Flags().StringVarP(&namespace, "namespace", "", "ecobee", "namespace to use in prometheus")
}

func promListen() {
	gaugesMap := make(map[string]prometheus.Gauge)
	gauges := []struct {
		name string
		val  func(ts *ecobee.ThermostatSummary, t *ecobee.Thermostat) float64
	}{
		{"fan", func(ts *ecobee.ThermostatSummary, t *ecobee.Thermostat) float64 {
			return boolToFloat(ts.EquipmentStatus.Fan)
		}},
		{"comp_cool1", func(ts *ecobee.ThermostatSummary, t *ecobee.Thermostat) float64 {
			return boolToFloat(ts.EquipmentStatus.CompCool1)
		}},
		{"comp_cool2", func(ts *ecobee.ThermostatSummary, t *ecobee.Thermostat) float64 {
			return boolToFloat(ts.EquipmentStatus.CompCool2)
		}},

		{"aux_heat1", func(ts *ecobee.ThermostatSummary, t *ecobee.Thermostat) float64 {
			return boolToFloat(ts.EquipmentStatus.AuxHeat1)
		}},
		{"aux_heat2", func(ts *ecobee.ThermostatSummary, t *ecobee.Thermostat) float64 {
			return boolToFloat(ts.EquipmentStatus.AuxHeat2)
		}},
		{"aux_heat3", func(ts *ecobee.ThermostatSummary, t *ecobee.Thermostat) float64 {
			return boolToFloat(ts.EquipmentStatus.AuxHeat3)
		}},

		{"desired_heat", func(ts *ecobee.ThermostatSummary, t *ecobee.Thermostat) float64 {
			return float64(t.Runtime.DesiredHeat) / 10.0
		}},
		{"desired_cool", func(ts *ecobee.ThermostatSummary, t *ecobee.Thermostat) float64 {
			return float64(t.Runtime.DesiredCool) / 10.0
		}},
		{"temperature", func(ts *ecobee.ThermostatSummary, t *ecobee.Thermostat) float64 {
			return float64(t.Runtime.ActualTemperature) / 10.0
		}},
	}
	for _, i := range gauges {
		g := prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      i.name,
				Help:      i.name,
			},
		)
		//g.Set(i.val)
		gaugesMap[i.name] = g
		prometheus.MustRegister(g)
	}

	sensorTemp := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "sensor_temperature",
			Help:      "Temperature",
		},
		[]string{"name"},
	)
	prometheus.MustRegister(sensorTemp)

	sensorOccupied := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "sensor_occupied",
			Help:      "Presense",
		},
		[]string{"name"},
	)
	prometheus.MustRegister(sensorOccupied)

	humidity := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "sensor_humidity",
			Help:      "Humidity",
		},
		[]string{"name"},
	)
	prometheus.MustRegister(humidity)

	cron := cron.New()
	cron.AddFunc("30 * * * *", func() {
		c := client()

		tsm, err := c.GetThermostatSummary(
			ecobee.Selection{
				SelectionType:          "thermostats",
				SelectionMatch:         thermostat,
				IncludeEquipmentStatus: true,
			})
		if err != nil {
			glog.Exitf("error retrieving thermostat summary for %s: %v", thermostat, err)
		}

		var ts ecobee.ThermostatSummary
		var ok bool

		if ts, ok = tsm[thermostat]; !ok {
			glog.Exitf("thermostat %s missing from ThermostatSummary", thermostat)
		}

		t, err := c.GetThermostat(thermostat)
		if err != nil {
			glog.Exitf("error retrieving thermostat %s: %v", thermostat, err)
		}

		for _, s := range t.RemoteSensors {
			for _, c := range s.Capability {
				if c.Type == "temperature" {
					t, err := strconv.ParseFloat(c.Value, 64)
					if err == nil {
						g, _ := sensorTemp.GetMetricWithLabelValues(s.Name)
						g.Set(t / 10.0)
					}
				}
				if c.Type == "occupancy" {
					g, _ := sensorOccupied.GetMetricWithLabelValues(s.Name)
					g.Set(stringBoolToFloat(c.Value))
				}
				if c.Type == "humidity" {
					t, err := strconv.ParseFloat(c.Value, 64)
					if err == nil {
						g, _ := humidity.GetMetricWithLabelValues(s.Name)
						g.Set(t)
					}
				}
			}
		}
		for _, i := range gauges {
			gaugesMap[i.name].Set(i.val(&ts, t))
		}
		log.Println("Collected info")
	})
	cron.Start()

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
