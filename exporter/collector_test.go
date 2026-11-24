// Copyright 2020 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package exporter

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/prometheus-community/json_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

func TestCollectMissingLabel(t *testing.T) {
	for _, scrapeType := range []config.ScrapeType{config.ValueScrape, config.ObjectScrape} {
		for _, tt := range []struct {
			name         string
			path         string
			allowMissing bool
			wantError    bool
		}{
			{"required", "{.missing}", false, true},
			{"optional", "{.missing}", true, false},
			{"invalid", "{.missing[}", true, true},
		} {
			t.Run(string(scrapeType)+"/"+tt.name, func(t *testing.T) {
				var logs bytes.Buffer
				metric := JSONMetric{
					Desc:            prometheus.NewDesc("test_value", "Test metric", []string{"optional", "present"}, nil),
					Type:            scrapeType,
					KeyJSONPath:     "{.value}",
					LabelsJSONPaths: []string{tt.path, "{.id}"},
					ValueType:       prometheus.GaugeValue,
					AllowMissingKey: tt.allowMissing,
				}
				if scrapeType == config.ObjectScrape {
					metric.KeyJSONPath = "{.objects[*]}"
					metric.ValueJSONPath = "{.value}"
				}
				collector := JSONMetricCollector{
					JSONMetrics: []JSONMetric{metric},
					Data:        []byte(`{"value":42,"id":"existing","objects":[{"value":42,"id":"existing"}]}`),
					Logger:      slog.New(slog.NewTextHandler(&logs, nil)),
				}
				registry := prometheus.NewPedanticRegistry()
				registry.MustRegister(collector)
				families, err := registry.Gather()
				if err != nil {
					t.Fatal(err)
				}
				if len(families) != 1 || len(families[0].Metric) != 1 {
					t.Fatalf("expected one metric, got %v", families)
				}
				got := families[0].Metric[0]
				if got.GetGauge().GetValue() != 42 {
					t.Errorf("expected value 42, got %v", got)
				}
				if len(got.Label) != 2 || got.Label[0].GetValue() != "" || got.Label[1].GetValue() != "existing" {
					t.Errorf("expected empty optional label and existing present label, got %v", got.Label)
				}
				if (logs.Len() > 0) != tt.wantError {
					t.Errorf("want error log %v, got %q", tt.wantError, logs.String())
				}
			})
		}
	}
}
