package domain

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func NormalizeCheckoutURL(value string) (string, error) {
	checkoutURL := strings.TrimSpace(value)
	if checkoutURL == "" {
		return "", errors.New("checkout URL is empty")
	}

	parsedCheckoutURL, err := url.Parse(checkoutURL)
	if err != nil {
		return "", fmt.Errorf("parse checkout URL: %w", err)
	}
	if strings.Contains(parsedCheckoutURL.Path, `\`) {
		return "", errors.New("checkout URL is unsafe")
	}
	if parsedCheckoutURL.IsAbs() {
		scheme := strings.ToLower(parsedCheckoutURL.Scheme)
		if (scheme != "http" && scheme != "https") ||
			parsedCheckoutURL.Host == "" ||
			parsedCheckoutURL.User != nil {
			return "", errors.New("checkout URL is unsafe")
		}
	} else if parsedCheckoutURL.Host != "" ||
		!strings.HasPrefix(parsedCheckoutURL.Path, "/") ||
		strings.HasPrefix(parsedCheckoutURL.Path, "//") {
		return "", errors.New("checkout URL is unsafe")
	}

	return checkoutURL, nil
}
