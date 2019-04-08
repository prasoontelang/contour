// Copyright © 2018 Heptio
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

package v1beta1

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

// IngressRouteSpec defines the spec of the CRD
type IngressRouteSpec struct {
	// Virtualhost appears at most once. If it is present, the object is considered
	// to be a "root".
	VirtualHost *VirtualHost `json:"virtualhost,omitempty"`
	// Routes are the ingress routes. If TCPProxy is present, Routes is ignored.
	Routes []Route `json:"routes"`
	// TCPProxy holds TCP proxy information.
	TCPProxy *TCPProxy `json:"tcpproxy,omitempty"`
}

// VirtualHost appears at most once. If it is present, the object is considered
// to be a "root".
type VirtualHost struct {
	// The fully qualified domain name of the root of the ingress tree
	// all leaves of the DAG rooted at this object relate to the fqdn
	Fqdn string `json:"fqdn"`
	// If present describes tls properties. The CNI names that will be matched on
	// are described in fqdn, the tls.secretName secret must contain a
	// matching certificate
	TLS *TLS `json:"tls,omitempty"`
	// retry policy for all the routes under this virtual host, unless the routes
	// have their own route policy defined
	RetryPolicy *RetryPolicy `json:"retryPolicy,omitempty"`
}

// TLS describes tls properties. The CNI names that will be matched on
// are described in fqdn, the tls.secretName secret must contain a
// matching certificate unless tls.passthrough is set to true.
type TLS struct {
	// required, the name of a secret in the current namespace
	SecretName string `json:"secretName,omitempty"`
	// Minimum TLS version this vhost should negotiate
	MinimumProtocolVersion string `json:"minimumProtocolVersion,omitempty"`
	// If Passthrough is set to true, the SecretName will be ignored
	// and the encrypted handshake will be passed through to the
	// backing cluster.
	Passthrough bool `json:"passthrough,omitempty"`
}

// Route contains the set of routes for a virtual host
type Route struct {
	// Match defines the prefix match
	Match string `json:"match"`
	// Services are the services to proxy traffic
	Services []Service `json:"services,omitempty"`
	// Delegate specifies that this route should be delegated to another IngressRoute
	Delegate *Delegate `json:"delegate,omitempty"`
	// Enables websocket support for the route
	EnableWebsockets bool `json:"enableWebsockets,omitempty"`
	// Allow this path to respond to insecure requests over HTTP which are normally
	// not permitted when a `virtualhost.tls` block is present.
	PermitInsecure bool `json:"permitInsecure,omitempty"`
	// Indicates that during forwarding, the matched prefix (or path) should be swapped with this value
	PrefixRewrite string `json:"prefixRewrite,omitempty"`
	// The request timeout for this route
	TimeoutPolicy *TimeoutPolicy `json:"timeoutPolicy,omitempty"`
	// The retry attempts for this route
	RetryPolicy *RetryPolicy `json:"retryPolicy,omitempty"`
}

// TimeoutPolicy defines the attributes associated with timeout
type TimeoutPolicy struct {
	// Timeout for establishing a connection in milliseconds
	Request *JsonDuration `json:"request"`
	// Timeout for receiving a response in seconds
	Idle *JsonDuration `json:"idle"`
}

// RetryPolicy defines the attributes associated with retrying policy
type RetryPolicy struct {
	// MaxRetries is maximum allowed number of retries
	NumRetries string `json:"count"`
	// Perform retry on failed requests with the matched status codes or aggregated as 5xx
	OnStatusCodes []string `json:"onStatusCodes"`
	// PerTryTimeout specifies the timeout per retry attempt. Ignored if OnStatusCodes are empty
	PerTryTimeout *JsonDuration `json:"perTryTimeout"`
}

// new struct to parse from JSON and load directly as time.Duration
type JsonDuration struct {
	*time.Duration
}

// to Unmarshal bytes of JSON into JsonDuration type
func (d *JsonDuration) UnmarshalJSON(b []byte) (err error) {

	timeStr := strings.Trim(string(b), `"`)
	duration, err := time.ParseDuration(timeStr)

	if err != nil && timeStr == "infinity" {
		duration = -1
		err = nil
	}

	if err == nil {
		// only if correctly parsed, we update the Duration. Else, Duration is initialized as nil
		d.Duration = &duration
	}
	return
}

// to Marshal JsonDuration into JSON bytes type (unused)
func (d JsonDuration) MarshalJSON() (b []byte, err error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

// a wrapper function to interpret all forms of input received from the YAML file for contour
func (d *JsonDuration) Time() (timeout time.Duration, valid bool) {
	if d == nil {
		// this means the timeout field (like request, idle, etc.) was not specified in the YAML file
		// not specifying the timeout field is a valid input
		timeout, valid = 0, true
	} else if d.Duration == nil {
		// this means the timeout field was incorrectly specified, hence it couldn't be casted to time.Duration
		// incorrectly parsed timeout is an invalid input
		timeout, valid = -1, false
	} else {
		// timeout was correctly parsed, hence valid input
		timeout, valid = *d.Duration, true
	}

	return
}

// TCPProxy contains the set of services to proxy TCP connections.
type TCPProxy struct {
	// Services are the services to proxy traffic
	Services []Service `json:"services,omitempty"`
	// Delegate specifies that this tcpproxy should be delegated to another IngressRoute
	Delegate *Delegate `json:"delegate,omitempty"`
}

// Service defines an upstream to proxy traffic to
type Service struct {
	// Name is the name of Kubernetes service to proxy traffic.
	// Names defined here will be used to look up corresponding endpoints which contain the ips to route.
	Name string `json:"name"`
	// Port (defined as Integer) to proxy traffic to since a service can have multiple defined
	Port int `json:"port"`
	// Weight defines percentage of traffic to balance traffic
	Weight int `json:"weight,omitempty"`
	// HealthCheck defines optional healthchecks on the upstream service
	HealthCheck *HealthCheck `json:"healthCheck,omitempty"`
	// LB Algorithm to apply (see https://github.com/heptio/contour/blob/master/design/ingressroute-design.md#load-balancing)
	Strategy string `json:"strategy,omitempty"`
}

// Delegate allows for delegating VHosts to other IngressRoutes
type Delegate struct {
	// Name of the IngressRoute
	Name string `json:"name"`
	// Namespace of the IngressRoute
	Namespace string `json:"namespace,omitempty"`
}

// HealthCheck defines optional healthchecks on the upstream service
type HealthCheck struct {
	// HTTP endpoint used to perform health checks on upstream service
	Path string `json:"path"`
	// The value of the host header in the HTTP health check request.
	// If left empty (default value), the name "contour-envoy-healthcheck"
	// will be used.
	Host string `json:"host,omitempty"`
	// The interval (seconds) between health checks
	IntervalSeconds int64 `json:"intervalSeconds"`
	// The time to wait (seconds) for a health check response
	TimeoutSeconds int64 `json:"timeoutSeconds"`
	// The number of unhealthy health checks required before a host is marked unhealthy
	UnhealthyThresholdCount uint32 `json:"unhealthyThresholdCount"`
	// The number of healthy health checks required before a host is marked healthy
	HealthyThresholdCount uint32 `json:"healthyThresholdCount"`
}

// Status reports the current state of the IngressRoute
type Status struct {
	CurrentStatus string `json:"currentStatus"`
	Description   string `json:"description"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IngressRoute is an Ingress CRD specificiation
type IngressRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   IngressRouteSpec `json:"spec"`
	Status `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IngressRouteList is a list of IngressRoutes
type IngressRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []IngressRoute `json:"items"`
}
