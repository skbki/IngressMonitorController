package uptimekuma

import (
	"reflect"
	"testing"

	endpointmonitorv1alpha1 "github.com/stakater/IngressMonitorController/v2/api/v1alpha1"
	"github.com/stakater/IngressMonitorController/v2/pkg/config"
	"github.com/stakater/IngressMonitorController/v2/pkg/models"
)

func TestProcessProviderConfig(t *testing.T) {
	service := &UptimeKumaMonitorService{}
	
	testCases := []struct {
		name     string
		monitor  models.Monitor
		expected map[string]interface{}
	}{
		{
			name: "Default configuration",
			monitor: models.Monitor{
				Name: "test-monitor",
				URL:  "https://example.com",
			},
			expected: map[string]interface{}{
				"name":               "test-monitor",
				"url":                "https://example.com",
				"type":               "http",
				"active":             true,
				"interval":           DefaultInterval,
				"timeout":            DefaultTimeout,
				"maxRedirects":       DefaultMaxRedirects,
				"method":             DefaultMethod,
				"expiryNotification": DefaultExpiryNotification,
			},
		},
		{
			name: "With provider configuration",
			monitor: models.Monitor{
				Name: "test-monitor",
				URL:  "https://example.com",
				Config: &endpointmonitorv1alpha1.UptimeKumaConfig{
					Interval:     120,
					Timeout:      30,
					MaxRedirects: 5,
					Method:       "POST",
					Body:         `{"test": "data"}`,
					Headers:      `{"Content-Type": "application/json"}`,
					BasicAuthUser: "testuser",
					BasicAuthPassword: "testpass",
					Keyword:      "success",
					InvertKeyword: true,
					IgnoreTls:    true,
					ExpiryNotification: 14,
					NotificationIDList: "1,2,3",
					ProxyId:      1,
					Tags:         "test,api",
				},
			},
			expected: map[string]interface{}{
				"name":               "test-monitor",
				"url":                "https://example.com",
				"type":               "http",
				"active":             true,
				"interval":           120,
				"timeout":            30,
				"maxRedirects":       5,
				"method":             "POST",
				"body":               `{"test": "data"}`,
				"headers":            `{"Content-Type": "application/json"}`,
				"basic_auth_user":    "testuser",
				"basic_auth_pass":    "testpass",
				"keyword":            "success",
				"invertKeyword":      true,
				"ignoreTls":          true,
				"expiryNotification": 14,
				"notificationIDList": map[string]bool{"1": true, "2": true, "3": true},
				"proxyId":            1,
				"tags":               []string{"test", "api"},
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := service.processProviderConfig(tc.monitor)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Expected %+v, got %+v", tc.expected, result)
			}
		})
	}
}

func TestSetup(t *testing.T) {
	service := &UptimeKumaMonitorService{}
	provider := config.Provider{
		ApiURL:   "https://uptime-kuma.example.com",
		Username: "admin",
		Password: "", // Empty password to avoid authentication call
	}
	
	service.Setup(provider)
	
	if service.apiURL != provider.ApiURL {
		t.Errorf("Expected apiURL to be %s, got %s", provider.ApiURL, service.apiURL)
	}
	
	if service.username != provider.Username {
		t.Errorf("Expected username to be %s, got %s", provider.Username, service.username)
	}
	
	if service.password != provider.Password {
		t.Errorf("Expected password to be %s, got %s", provider.Password, service.password)
	}
}

func TestMapToBaseMonitor(t *testing.T) {
	service := &UptimeKumaMonitorService{}
	
	monitor := Monitor{
		ID:   123,
		Name: "test-monitor",
		URL:  "https://example.com",
	}
	
	result := service.mapToBaseMonitor(monitor)
	
	if result.ID != "123" {
		t.Errorf("Expected ID to be '123', got '%s'", result.ID)
	}
	
	if result.Name != "test-monitor" {
		t.Errorf("Expected Name to be 'test-monitor', got '%s'", result.Name)
	}
	
	if result.URL != "https://example.com" {
		t.Errorf("Expected URL to be 'https://example.com', got '%s'", result.URL)
	}
}