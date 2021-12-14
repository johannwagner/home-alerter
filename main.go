package main

import (
	"errors"
	"fmt"
	"github.com/johannwagner/home-alerter/alertmanager"
	"github.com/johannwagner/home-alerter/telegram"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"os"
	"strconv"
	"time"
)

func diffAlertLists(a, b []*alertmanager.TriggeredAlert) []string {
	var diff []string

	// Loop two times, first to find slice1 strings not in slice2,
	// second loop to find slice2 strings not in slice1
	for i := 0; i < 2; i++ {
		for _, s1 := range a {
			found := false
			for _, s2 := range b {
				if s1.Key == s2.Key {
					found = true
					break
				}
			}
			// String not found. We add it to return slice
			if !found {
				diff = append(diff, s1.Key)
			}
		}
		// Swap the slices, only if it was the first loop
		if i == 0 {
			a, b = b, a
		}
	}

	return diff
}

func main() {

	envChatId, hasChatId := os.LookupEnv("TELEGRAM_CHAT_ID")
	envBotToken, hasBotToken := os.LookupEnv("TELEGRAM_BOT_TOKEN")
	envEndpoint, hasMetricsEndpoint := os.LookupEnv("METRICS_ENDPOINT")

	chatId, err := strconv.Atoi(envChatId)

	if err != nil || !hasChatId || !hasBotToken || !hasMetricsEndpoint {
		panic(errors.New("Invalid Configuration"))
	}

	alertManager := alertmanager.NewAlertManager()

	alertManager.AddRule(
		"Heizleistung",
		"Die Heizleistung im Zimmer *{{ .zone }}* ist über 60%.",
		"tado_activity_heating_power_percentage",
		func(m *io_prometheus_client.Metric) bool {
			var maximumHeat = float64(60)
			return *m.Gauge.Value > maximumHeat
		},
	)

	alertManager.AddRule(
		"Feuchtigkeit",
		"Die Luftfeuchtigkeit im Zimmer *{{ .zone }}* ist über 75%.",
		"tado_sensor_humidity_percentage",
		func(m *io_prometheus_client.Metric) bool {
			var maximumHumidity = float64(75)
			return *m.Gauge.Value > maximumHumidity
		},
	)

	alertManager.AddRule(
		"Temperatur",
		"Die Temperatur im Zimmer *{{ .zone }}* ist unter 16 Grad.",
		"tado_sensor_temperature_value",
		func(m *io_prometheus_client.Metric) bool {

			invalidMetric := false
			// Check for fahrenheit unit
			for _, pair := range m.Label {
				if pair.GetName() == "unit" && pair.GetValue() != "celsius" {
					invalidMetric = true
				}
			}

			if invalidMetric {
				return false
			}

			var minimumTemperatur = float64(16)
			return *m.Gauge.Value < minimumTemperatur
		},
	)

	telegramBot, err := telegram.New(envBotToken, int64(chatId))
	if err != nil {
		panic(err)
	}

	ticker := time.NewTicker(1 * time.Minute)

	savedAlerts := []*alertmanager.TriggeredAlert{}

	for {
		select {
		case <-ticker.C:
			fmt.Printf("Checking for alerts\n")

			triggeredAlerts, err := alertManager.CheckEndpoint(envEndpoint)
			if err != nil {
				panic(err)
			}

			fmt.Printf("Found %v alerts\n", len(triggeredAlerts))

			diffAlertList := diffAlertLists(savedAlerts, triggeredAlerts)

			if len(diffAlertList) > 0 {

				fmt.Printf("Found other alerts, sending message\n")
				err = telegramBot.WriteMessage(triggeredAlerts)

				if err != nil {
					panic(err)
				}
			}

			savedAlerts = triggeredAlerts
		}
	}

}
