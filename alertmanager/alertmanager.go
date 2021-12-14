package alertmanager

import (
	"github.com/google/uuid"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"net/http"
	"strings"
)

type TriggeredAlert struct {
	Key    string
	Rule   *AlertRule
	Metric *io_prometheus_client.Metric
}

func NewTriggeredAlert(rule *AlertRule, metric *io_prometheus_client.Metric) *TriggeredAlert {
	keyParts := []string{rule.Metric}

	for _, pair := range metric.Label {
		keyParts = append(keyParts, pair.GetName(), pair.GetValue())
	}

	key := strings.Join(keyParts, "_")

	triggeredAlert := TriggeredAlert{
		key,
		rule,
		metric,
	}

	return &triggeredAlert
}

type AlertRule struct {
	UUID           uuid.UUID
	Title          string
	Description    string
	Metric         string
	EvaluationFunc func(metric *io_prometheus_client.Metric) bool
}

type AlertManager struct {
	rules []*AlertRule
}

func NewAlertManager() *AlertManager {
	alertManager := AlertManager{}
	return &alertManager
}

func (am *AlertManager) AddRule(title string, description string, metric string, evaluationFunc func(*io_prometheus_client.Metric) bool) {
	alertUUID, _ := uuid.NewUUID()
	rule := AlertRule{
		alertUUID,
		title,
		description,
		metric,
		evaluationFunc,
	}
	am.rules = append(am.rules, &rule)
}

func (am *AlertManager) CheckEndpoint(url string) ([]*TriggeredAlert, error) {

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var parser expfmt.TextParser
	metricsFamily, err := parser.TextToMetricFamilies(resp.Body)

	var triggeredAlerts []*TriggeredAlert

	for _, rule := range am.rules {
		metricsContainer := metricsFamily[rule.Metric]
		metrics := metricsContainer.GetMetric()

		for _, metric := range metrics {
			shouldAlert := rule.EvaluationFunc(metric)

			if shouldAlert {
				triggeredAlert := NewTriggeredAlert(rule, metric)
				triggeredAlerts = append(triggeredAlerts, triggeredAlert)
			}
		}
	}

	return triggeredAlerts, nil
}
