package avitoadapter

import (
	"fmt"
	"net/http"
	"net/url"

	generated "github.com/avito-hack/queue/services/tickets/gen/clients/avitoadapter"
)

func newGeneratedClient(baseURL string, client *http.Client) (*generated.ClientWithResponses, error) {
	if client == nil {
		return nil, fmt.Errorf("http client is nil")
	}

	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse avito adapter URL: %w", err)
	}
	if parsedBaseURL.Scheme != "http" && parsedBaseURL.Scheme != "https" {
		return nil, fmt.Errorf("avito adapter URL must use HTTP or HTTPS")
	}
	if parsedBaseURL.Host == "" {
		return nil, fmt.Errorf("avito adapter URL host is required")
	}
	if parsedBaseURL.RawQuery != "" || parsedBaseURL.Fragment != "" {
		return nil, fmt.Errorf("avito adapter URL must not contain query or fragment")
	}

	clientCopy := *client
	clientCopy.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	generatedClient, err := generated.NewClientWithResponses(
		parsedBaseURL.String(),
		generated.WithHTTPClient(&clientCopy),
	)
	if err != nil {
		return nil, fmt.Errorf("create generated avito adapter client: %w", err)
	}

	return generatedClient, nil
}
