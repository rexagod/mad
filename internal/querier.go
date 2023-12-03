package internal

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

const (

	// tokenSAPath is the path to the service account token.
	tokenSAPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

	// caCertSAPath is the path to the service account CA certificate.
	certSAPath = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

// Querier knows how to query the endpoints of all associated components.
type Querier struct {

	// client is the http client used to query the endpoints.
	client *http.Client

	// token is the service account token.
	token string
}

// NewQuerier creates a new Querier.
func NewQuerier() *Querier {
	utilruntime.HandleCrash()

	// Read the SA token.
	token, err := os.ReadFile(tokenSAPath)
	if err != nil {
		panic(err)
	}

	// Read the CA certificate.
	caCert, err := os.ReadFile(certSAPath)
	if err != nil {
		panic(err)
	}
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caCert)

	// Create the client.
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certPool,
			},
		},
	}

	// Create the Querier.
	querier := &Querier{
		client: client,
		token:  string(token),
	}

	return querier
}

// DoMADQuery queries the healthcheck endpoint.
func (q *Querier) DoMADQuery(endpoint string) bool {

	// Create the request.
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return false
	}

	// Add the token to the request.
	req.Header.Add("Authorization", "Bearer "+q.token)

	// Perform the request.
	resp, err := q.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Check the response.
	return resp.StatusCode == http.StatusOK
}
