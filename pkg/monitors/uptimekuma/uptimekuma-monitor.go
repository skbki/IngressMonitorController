package uptimekuma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	endpointmonitorv1alpha1 "github.com/stakater/IngressMonitorController/v2/api/v1alpha1"
	"github.com/stakater/IngressMonitorController/v2/pkg/config"
	httpClient "github.com/stakater/IngressMonitorController/v2/pkg/http"
	"github.com/stakater/IngressMonitorController/v2/pkg/models"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("uptimekuma-monitor")

// UptimeKumaMonitorService struct contains parameters required for Uptime Kuma
type UptimeKumaMonitorService struct {
	apiURL    string
	username  string
	password  string
	authToken string
}

// Default values for Uptime Kuma monitor
const (
	DefaultInterval      = 60  // 60 seconds
	DefaultTimeout       = 48  // 48 seconds
	DefaultMaxRedirects  = 10
	DefaultMethod        = "GET"
	DefaultExpiryNotification = 7 // 7 days
)

// Monitor represents an Uptime Kuma monitor
type Monitor struct {
	ID                 int               `json:"id,omitempty"`
	Name               string            `json:"name"`
	URL                string            `json:"url"`
	Type               string            `json:"type"`
	Interval           int               `json:"interval"`
	Timeout            int               `json:"timeout"`
	MaxRedirects       int               `json:"maxRedirects"`
	Method             string            `json:"method"`
	Body               string            `json:"body,omitempty"`
	Headers            string            `json:"headers,omitempty"`
	BasicAuthUser      string            `json:"basic_auth_user,omitempty"`
	BasicAuthPass      string            `json:"basic_auth_pass,omitempty"`
	Keyword            string            `json:"keyword,omitempty"`
	InvertKeyword      bool              `json:"invertKeyword,omitempty"`
	IgnoreTls          bool              `json:"ignoreTls,omitempty"`
	ExpiryNotification int               `json:"expiryNotification,omitempty"`
	NotificationIDList map[string]bool   `json:"notificationIDList,omitempty"`
	ProxyId            int               `json:"proxyId,omitempty"`
	Tags               []string          `json:"tags,omitempty"`
	Active             bool              `json:"active"`
}

// APIResponse represents a generic Uptime Kuma API response
type APIResponse struct {
	OK  bool   `json:"ok"`
	Msg string `json:"msg,omitempty"`
}

// MonitorResponse represents the response when creating/updating a monitor
type MonitorResponse struct {
	APIResponse
	MonitorID int `json:"monitorID,omitempty"`
}

// MonitorListResponse represents the response when getting monitors
type MonitorListResponse struct {
	APIResponse
	Monitors []Monitor `json:"monitors,omitempty"`
}

func (service *UptimeKumaMonitorService) Equal(oldMonitor models.Monitor, newMonitor models.Monitor) bool {
	oldConfig := service.processProviderConfig(oldMonitor)
	newConfig := service.processProviderConfig(newMonitor)
	
	if !reflect.DeepEqual(oldConfig, newConfig) {
		log.Info(fmt.Sprintf("There are some new changes in %s monitor", newMonitor.Name))
		return false
	}
	return true
}

func (service *UptimeKumaMonitorService) Setup(p config.Provider) {
	service.apiURL = p.ApiURL
	service.username = p.Username
	service.password = p.Password
	
	// Only authenticate if we have credentials and a URL
	if service.apiURL != "" && service.username != "" && service.password != "" {
		err := service.authenticate()
		if err != nil {
			log.Error(err, "Failed to authenticate with Uptime Kuma")
		}
	}
}

func (service *UptimeKumaMonitorService) GetByName(name string) (*models.Monitor, error) {
	monitors := service.GetAll()
	
	for _, monitor := range monitors {
		if monitor.Name == name {
			return &monitor, nil
		}
	}
	
	return nil, nil
}

func (service *UptimeKumaMonitorService) GetAll() []models.Monitor {
	client := httpClient.CreateHttpClient(service.apiURL + "/api/monitors")
	headers := service.getAuthHeaders()
	
	response := client.GetUrl(headers, nil)
	
	if response.StatusCode != http.StatusOK {
		log.Error(fmt.Errorf("API request failed with status %d", response.StatusCode), "Failed to get monitors")
		return []models.Monitor{}
	}
	
	var apiResponse MonitorListResponse
	err := json.Unmarshal(response.Bytes, &apiResponse)
	if err != nil {
		log.Error(err, "Failed to unmarshal response")
		return []models.Monitor{}
	}
	
	if !apiResponse.OK {
		log.Error(fmt.Errorf("API returned error: %s", apiResponse.Msg), "Failed to get monitors")
		return []models.Monitor{}
	}
	
	var monitors []models.Monitor
	for _, monitor := range apiResponse.Monitors {
		monitors = append(monitors, service.mapToBaseMonitor(monitor))
	}
	
	return monitors
}

func (service *UptimeKumaMonitorService) Add(m models.Monitor) {
	monitorData := service.processProviderConfig(m)
	
	jsonData, err := json.Marshal(monitorData)
	if err != nil {
		log.Error(err, "Failed to marshal monitor data")
		return
	}
	
	client := httpClient.CreateHttpClient(service.apiURL + "/api/monitor")
	headers := service.getAuthHeaders()
	headers["Content-Type"] = "application/json"
	
	response := client.PostUrl(headers, jsonData)
	
	if response.StatusCode == http.StatusOK {
		var apiResponse MonitorResponse
		err := json.Unmarshal(response.Bytes, &apiResponse)
		if err != nil {
			log.Error(err, "Failed to unmarshal response")
			return
		}
		
		if apiResponse.OK {
			log.Info(fmt.Sprintf("Monitor Added: %s (ID: %d)", m.Name, apiResponse.MonitorID))
		} else {
			log.Error(fmt.Errorf("API returned error: %s", apiResponse.Msg), "Failed to add monitor")
		}
	} else {
		log.Error(fmt.Errorf("API request failed with status %d", response.StatusCode), "Failed to add monitor")
	}
}

func (service *UptimeKumaMonitorService) Update(m models.Monitor) {
	// Get the existing monitor to find its ID
	existingMonitor, err := service.GetByName(m.Name)
	if err != nil || existingMonitor == nil {
		log.Error(fmt.Errorf("monitor not found: %s", m.Name), "Failed to update monitor")
		return
	}
	
	monitorData := service.processProviderConfig(m)
	monitorData["id"] = existingMonitor.ID
	
	jsonData, err := json.Marshal(monitorData)
	if err != nil {
		log.Error(err, "Failed to marshal monitor data")
		return
	}
	
	client := httpClient.CreateHttpClient(service.apiURL + "/api/monitor/" + existingMonitor.ID)
	headers := service.getAuthHeaders()
	headers["Content-Type"] = "application/json"
	
	response := client.PutUrl(headers, jsonData)
	
	if response.StatusCode == http.StatusOK {
		var apiResponse APIResponse
		err := json.Unmarshal(response.Bytes, &apiResponse)
		if err != nil {
			log.Error(err, "Failed to unmarshal response")
			return
		}
		
		if apiResponse.OK {
			log.Info(fmt.Sprintf("Monitor Updated: %s", m.Name))
		} else {
			log.Error(fmt.Errorf("API returned error: %s", apiResponse.Msg), "Failed to update monitor")
		}
	} else {
		log.Error(fmt.Errorf("API request failed with status %d", response.StatusCode), "Failed to update monitor")
	}
}

func (service *UptimeKumaMonitorService) Remove(m models.Monitor) {
	// Get the existing monitor to find its ID
	existingMonitor, err := service.GetByName(m.Name)
	if err != nil || existingMonitor == nil {
		log.Error(fmt.Errorf("monitor not found: %s", m.Name), "Failed to remove monitor")
		return
	}
	
	client := httpClient.CreateHttpClient(service.apiURL + "/api/monitor/" + existingMonitor.ID)
	headers := service.getAuthHeaders()
	
	response := client.DeleteUrl(headers, nil)
	
	if response.StatusCode == http.StatusOK {
		var apiResponse APIResponse
		err := json.Unmarshal(response.Bytes, &apiResponse)
		if err != nil {
			log.Error(err, "Failed to unmarshal response")
			return
		}
		
		if apiResponse.OK {
			log.Info(fmt.Sprintf("Monitor Removed: %s", m.Name))
		} else {
			log.Error(fmt.Errorf("API returned error: %s", apiResponse.Msg), "Failed to remove monitor")
		}
	} else {
		log.Error(fmt.Errorf("API request failed with status %d", response.StatusCode), "Failed to remove monitor")
	}
}

func (service *UptimeKumaMonitorService) authenticate() error {
	loginData := map[string]string{
		"username": service.username,
		"password": service.password,
	}
	
	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return err
	}
	
	client := httpClient.CreateHttpClient(service.apiURL + "/api/login")
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	
	response := client.PostUrl(headers, jsonData)
	
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed with status %d", response.StatusCode)
	}
	
	var authResponse struct {
		OK    bool   `json:"ok"`
		Token string `json:"token,omitempty"`
		Msg   string `json:"msg,omitempty"`
	}
	
	err = json.Unmarshal(response.Bytes, &authResponse)
	if err != nil {
		return err
	}
	
	if !authResponse.OK {
		return fmt.Errorf("authentication failed: %s", authResponse.Msg)
	}
	
	service.authToken = authResponse.Token
	return nil
}

func (service *UptimeKumaMonitorService) getAuthHeaders() map[string]string {
	headers := make(map[string]string)
	if service.authToken != "" {
		headers["Authorization"] = "Bearer " + service.authToken
	}
	return headers
}

func (service *UptimeKumaMonitorService) processProviderConfig(m models.Monitor) map[string]interface{} {
	providerConfig, _ := m.Config.(*endpointmonitorv1alpha1.UptimeKumaConfig)
	
	config := make(map[string]interface{})
	config["name"] = m.Name
	config["url"] = m.URL
	config["type"] = "http"
	config["active"] = true
	
	// Set defaults and override with provider config
	config["interval"] = DefaultInterval
	config["timeout"] = DefaultTimeout
	config["maxRedirects"] = DefaultMaxRedirects
	config["method"] = DefaultMethod
	config["expiryNotification"] = DefaultExpiryNotification
	
	if providerConfig != nil {
		if providerConfig.Interval > 0 {
			config["interval"] = providerConfig.Interval
		}
		if providerConfig.Timeout > 0 {
			config["timeout"] = providerConfig.Timeout
		}
		if providerConfig.MaxRedirects >= 0 {
			config["maxRedirects"] = providerConfig.MaxRedirects
		}
		if providerConfig.Method != "" {
			config["method"] = providerConfig.Method
		}
		if providerConfig.Body != "" {
			config["body"] = providerConfig.Body
		}
		if providerConfig.Headers != "" {
			config["headers"] = providerConfig.Headers
		}
		if providerConfig.BasicAuthUser != "" {
			config["basic_auth_user"] = providerConfig.BasicAuthUser
		}
		if providerConfig.BasicAuthPassword != "" {
			config["basic_auth_pass"] = providerConfig.BasicAuthPassword
		}
		if providerConfig.Keyword != "" {
			config["keyword"] = providerConfig.Keyword
		}
		config["invertKeyword"] = providerConfig.InvertKeyword
		config["ignoreTls"] = providerConfig.IgnoreTls
		if providerConfig.ExpiryNotification > 0 {
			config["expiryNotification"] = providerConfig.ExpiryNotification
		}
		if providerConfig.NotificationIDList != "" {
			// Parse notification ID list (comma-separated string to map)
			notificationIDs := strings.Split(providerConfig.NotificationIDList, ",")
			notificationMap := make(map[string]bool)
			for _, id := range notificationIDs {
				id = strings.TrimSpace(id)
				if id != "" {
					notificationMap[id] = true
				}
			}
			config["notificationIDList"] = notificationMap
		}
		if providerConfig.ProxyId > 0 {
			config["proxyId"] = providerConfig.ProxyId
		}
		if providerConfig.Tags != "" {
			// Parse tags (comma-separated string to array)
			tags := strings.Split(providerConfig.Tags, ",")
			var tagArray []string
			for _, tag := range tags {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					tagArray = append(tagArray, tag)
				}
			}
			config["tags"] = tagArray
		}
	}
	
	return config
}

func (service *UptimeKumaMonitorService) mapToBaseMonitor(monitor Monitor) models.Monitor {
	return models.Monitor{
		ID:   strconv.Itoa(monitor.ID),
		Name: monitor.Name,
		URL:  monitor.URL,
	}
}