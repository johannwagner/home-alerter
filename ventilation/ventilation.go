package ventilation

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/common/expfmt"
	"io"
	"math"
	"net/http"
)

type VentilationRecommendation string

const (
	VentNo    VentilationRecommendation = "no"
	VentMaybe VentilationRecommendation = "maybe"
	VentYes   VentilationRecommendation = "yes"
)

type OWMResponse struct {
	Coord struct {
		Lon float64 `json:"lon"`
		Lat float64 `json:"lat"`
	} `json:"coord"`
	Weather []struct {
		ID          int    `json:"id"`
		Main        string `json:"main"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	} `json:"weather"`
	Base string `json:"base"`
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		TempMin   float64 `json:"temp_min"`
		TempMax   float64 `json:"temp_max"`
		Pressure  int     `json:"pressure"`
		Humidity  int     `json:"humidity"`
	} `json:"main"`
	Visibility int `json:"visibility"`
	Wind       struct {
		Speed float64 `json:"speed"`
		Deg   int     `json:"deg"`
	} `json:"wind"`
	Clouds struct {
		All int `json:"all"`
	} `json:"clouds"`
	Dt  int `json:"dt"`
	Sys struct {
		Type    int    `json:"type"`
		ID      int    `json:"id"`
		Country string `json:"country"`
		Sunrise int    `json:"sunrise"`
		Sunset  int    `json:"sunset"`
	} `json:"sys"`
	Timezone int    `json:"timezone"`
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Cod      int    `json:"cod"`
}

type VentilationManager struct {
	metricsUrl           string
	locLatitude          string
	locLongitude         string
	openWeatherMapApiKey string
}

func NewVentilationManager(metricsUrl string, locLatitude string, locLongitude string, openWeatherMapApiKey string) (*VentilationManager, error) {

	vM := VentilationManager{
		metricsUrl:           metricsUrl,
		locLatitude:          locLatitude,
		locLongitude:         locLongitude,
		openWeatherMapApiKey: openWeatherMapApiKey,
	}

	return &vM, nil
}

func GetAbsoluteHumidity(T float64, rh float64) float64 {
	return (6.112 * math.Pow(math.E, (17.67*T)/(T+243.5)) * rh * 2.1674) / (273.15 + T)

}

func (v *VentilationManager) GetOpenWeatherInformation() (float64, int, error) {
	metricsUrl := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?lat=%v&lon=%v&appid=%v", v.locLatitude, v.locLongitude, v.openWeatherMapApiKey)
	resp, err := http.Get(metricsUrl)
	if err != nil {
		return 0, 0, err
	}

	defer resp.Body.Close()
	respString, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, err
	}

	var sResp OWMResponse
	err = json.Unmarshal(respString, &sResp)
	if err != nil {
		return 0, 0, err
	}

	return sResp.Main.Temp - 273.15, sResp.Main.Humidity, nil
}

func (v *VentilationManager) GetAverageRoomStats() (float64, float64, error) {
	resp, err := http.Get(v.metricsUrl)
	if err != nil {
		return 0, 0, err
	}

	defer resp.Body.Close()

	var parser expfmt.TextParser
	metricsFamily, err := parser.TextToMetricFamilies(resp.Body)

	if err != nil {
		return 0, 0, err
	}

	temperaturValues := metricsFamily["tado_sensor_temperature_value"]
	humidityValues := metricsFamily["tado_sensor_humidity_percentage"]

	avgTempSum := float64(0)
	avgTempC := 0

	for _, m := range temperaturValues.Metric {
		labelMap := map[string]string{}
		for _, l := range m.Label {
			labelMap[l.GetName()] = l.GetValue()
		}

		if labelMap["unit"] != "celsius" || labelMap["type"] != "HEATING" {
			continue
		}
		avgTempSum += m.Gauge.GetValue()
		avgTempC += 1
	}

	avgHumSum := float64(0)
	avgHumC := 0

	for _, m := range humidityValues.Metric {
		labelMap := map[string]string{}
		for _, l := range m.Label {
			labelMap[l.GetName()] = l.GetValue()
		}

		if labelMap["type"] != "HEATING" {
			continue
		}
		avgHumSum += m.Gauge.GetValue()
		avgHumC += 1
	}

	avgTemp := avgTempSum / float64(avgTempC)
	avgHum := avgHumSum / float64(avgHumC)

	return avgTemp, avgHum, nil
}

func (v *VentilationManager) NeedsVentilation() (VentilationRecommendation, error) {
	outdoorTemp, outdoorHum, err := v.GetOpenWeatherInformation()
	if err != nil {
		return VentNo, err
	}

	indoorTemp, indoorHum, err := v.GetAverageRoomStats()
	if err != nil {
		return VentNo, err
	}

	outdoorAbsHum := GetAbsoluteHumidity(outdoorTemp, float64(outdoorHum))
	indoorAbsHum := GetAbsoluteHumidity(indoorTemp, float64(indoorHum))

	humidityDiff := indoorAbsHum - outdoorAbsHum

	if humidityDiff < 0.5 {
		return VentNo, nil
	} else if humidityDiff < 2 {
		return VentMaybe, nil
	} else {
		return VentYes, nil
	}
}
