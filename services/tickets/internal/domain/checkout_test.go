package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeCheckoutURL_SafeURL_ReturnNormalizedURL(t *testing.T) {
	// given
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{name: "same origin path", value: "  /checkout/1#payment  ", expected: "/checkout/1#payment"},
		{name: "HTTPS URL", value: "https://www.avito.ru/checkout/1", expected: "https://www.avito.ru/checkout/1"},
		{name: "HTTP URL", value: "http://avito.test/checkout/1", expected: "http://avito.test/checkout/1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// when
			result, err := NormalizeCheckoutURL(test.value)

			// then
			require.NoError(t, err)
			require.Equal(t, test.expected, result)
		})
	}
}

func TestNormalizeCheckoutURL_UnsafeURL_ReturnError(t *testing.T) {
	// given
	tests := []string{
		"",
		"javascript:alert(1)",
		"//example.com/checkout",
		"///example.com/checkout",
		"checkout/1",
		`/\example.com/checkout`,
		"https://user:password@example.com/checkout",
		"https://",
	}

	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			// when
			_, err := NormalizeCheckoutURL(value)

			// then
			require.Error(t, err)
		})
	}
}
