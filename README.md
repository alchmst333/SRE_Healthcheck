# Fetch Reward SRE Take Home Assignment

## Endpoint Health Checker

The Endpoint Health Checker is a simple Go-based tool designed for internal use to monitor the availability and performance of web services. It periodically sends HTTP requests to endpoints defined in a YAML configuration file, logs the results, and provides insights into the uptime and response times of these services.

## How It Works

- Periodically checks the status of configured endpoints.
- Tracks success, failure, and latency metrics for each service.
- Logs the results into a file for future analysis.
- Displays availability statistics in the terminal.

## Prerequisites

- Go (version =>1.16)
- YAML configuration file specifying endpoints to be monitored

## How to Use

1. Clone the Repository.

````bash
 git clone https://your-internal-repo/endpoint-health-checker.git
 cd endpoint-health-checker
 ````

2. Dependency Management.

````bash
go mod tidy
````

3. Build the Application.

````bash
go build -o healthchecker main.go
````

4. Create a YAML Configuration File.

- The YAML file defines the services needed for monitoring. The program requires the following fields: Name, URL. The following fields are optional: Method, Headers. Review provided sample YAML configuration file for formatting structure.
- Example config.yaml structure:

````bash
- name: Internal API
url: https://internal-api.yourcompany.com/health
method: GET
- name: Another Service
url: https://service.yourcompany.com/status
method: POST
headers:
````

1. Run the Health Checker

- Run the application with your configuration file.

````bash
# Simple run using default configurations. 
./healthchecker --file=path-to-file.yaml

# Customized run.
./healthchecker --file=path-to-file.yaml --log=logFile-name.log --interval=30s --latency=500ms
````

- Command-Line Flags
- --file: Path to the YAML config file (default: ./sample-input.yaml).
- --log: Path to the log file (default: ./healthcheck.log).
- --interval: Interval between checks (default: 15s).
- --latency: Maximum allowed latency for a successful check (default: 500ms).

6. Monitor Results

- Console Output: Shows availability percentages and latency metrics.
- Log file: Logs detailed log information about each health check in the specified log file.  

### Additional Enhancements and Recommendations

- Retry Logic for transient failures before marking an endpoint as DOWN.
- Integrate with Alerting Mechanisms to send notifications for endpoint status changes.  Examples: email, slack, etc
- Dashboard Instrumentation for metrics visualization. Example: Prometheus w/ Grafana
