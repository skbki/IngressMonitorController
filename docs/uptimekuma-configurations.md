# Uptime Kuma Configuration

## Prerequisites

To use Uptime Kuma with Ingress Monitor Controller, you need:

1. A running Uptime Kuma instance
2. Admin credentials for the Uptime Kuma instance
3. The API URL of your Uptime Kuma instance

## Configuration

The Uptime Kuma monitor provider supports various configuration options through the `uptimeKumaConfig` field:

|                        Fields                    |                    Description                               | Default |
|:----------------------------------------------------:|:------------------------------------------------------------:|:-------:|
| interval            | The check interval in seconds (minimum 20)             | 60      |
| timeout             | Timeout in seconds                                      | 48      |
| maxRedirects        | Maximum number of redirects to follow (0 to disable)   | 10      |
| method              | Request method (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS) | GET     |
| body                | Request body (for POST, PUT, PATCH requests)           | -       |
| headers             | Request headers in JSON format                          | -       |
| basicAuthUser       | Basic authentication username                           | -       |
| basicAuthPassword   | Basic authentication password (environment variable name) | -    |
| keyword             | Keywords to check in response                           | -       |
| invertKeyword       | Invert keyword check                                    | false   |
| ignoreTls           | Ignore TLS/SSL errors                                   | false   |
| expiryNotification  | Certificate expiry notification (days before expiry)   | 7       |
| notificationIDList  | Push notification settings (comma-separated IDs)       | -       |
| proxyId             | Proxy settings (proxy ID)                               | -       |
| tags                | Tags for the monitor (comma-separated)                  | -       |

## Provider Configuration

Add Uptime Kuma as a provider in your configuration:

```yaml
providers:
  - name: UptimeKuma
    apiURL: https://your-uptime-kuma-instance.com
    username: admin
    password: your-password
```

## Example Usage

### Basic HTTP Monitor

```yaml
apiVersion: endpointmonitor.stakater.com/v1alpha1
kind: EndpointMonitor
metadata:
  name: stakater-website
spec:
  forceHttps: true
  url: https://stakater.com/
  providers: UptimeKuma
  uptimeKumaConfig:
    interval: 120
    timeout: 30
```

### Advanced HTTP Monitor with Authentication

```yaml
apiVersion: endpointmonitor.stakater.com/v1alpha1
kind: EndpointMonitor
metadata:
  name: api-monitor
spec:
  url: https://api.example.com/health
  providers: UptimeKuma
  uptimeKumaConfig:
    interval: 60
    timeout: 30
    method: POST
    body: '{"check": "health"}'
    headers: '{"Content-Type": "application/json", "Authorization": "Bearer token"}'
    basicAuthUser: apiuser
    basicAuthPassword: API_PASSWORD_ENV_VAR
    keyword: "healthy"
    ignoreTls: false
    tags: "api,health,production"
    notificationIDList: "1,2,3"
```

### Monitor with Custom Settings

```yaml
apiVersion: endpointmonitor.stakater.com/v1alpha1
kind: EndpointMonitor
metadata:
  name: custom-monitor
spec:
  url: https://example.com
  providers: UptimeKuma
  uptimeKumaConfig:
    interval: 300  # 5 minutes
    timeout: 60
    maxRedirects: 5
    method: GET
    keyword: "success"
    invertKeyword: false
    ignoreTls: true
    expiryNotification: 14  # 14 days before cert expiry
    proxyId: 1
    tags: "website,monitoring"
```

## Notes

- The minimum interval is 20 seconds as enforced by Kubernetes validation
- Basic authentication password should reference an environment variable name for security
- Headers should be provided as a JSON string
- Notification IDs should be comma-separated and correspond to existing notification channels in your Uptime Kuma instance
- Tags are comma-separated and will be applied to the monitor in Uptime Kuma
- The proxy ID should correspond to an existing proxy configuration in your Uptime Kuma instance