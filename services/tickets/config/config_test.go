package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_ReturnConfig(t *testing.T) {
	// given
	setValidEnvironment(t)

	// when
	config, err := Load()

	// then
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1", config.HTTP.Host)
	assert.Equal(t, 8090, config.HTTP.Port)
	assert.Equal(t, 2*time.Second, config.HTTP.ReadTimeout)
	assert.Equal(t, 4*time.Second, config.HTTP.WriteTimeout)
	assert.Equal(t, 6*time.Second, config.HTTP.ShutdownTimeout)
	assert.Equal(t, "postgres://tickets:password@postgres/tickets", config.PostgreSQL.URL)
	assert.Equal(t, 2*time.Second, config.PostgreSQL.ConnectTimeout)
	assert.Equal(t, "http://avito-adapter:8080", config.AvitoAdapter.URL)
	assert.Equal(t, 3*time.Second, config.AvitoAdapter.Timeout)
	assert.Equal(t, "amqp://tickets:password@rabbitmq:5672/", config.RabbitMQ.URL)
	assert.Equal(t, 12*time.Minute, config.Ticket.ActivationTTL)
	assert.Equal(t, "queue-to-tickets-secret", config.ServiceAuthToken)
}

func TestLoad_DefaultTicketActivationTTL_ReturnFifteenMinutes(t *testing.T) {
	// given
	setValidEnvironment(t)
	require.NoError(t, os.Unsetenv("TICKET_ACTIVATION_TTL"))

	// when
	config, err := Load()

	// then
	require.NoError(t, err)
	assert.Equal(t, 15*time.Minute, config.Ticket.ActivationTTL)
}

func TestLoad_InvalidDependencyTimeout_ReturnError(t *testing.T) {
	// given
	tests := []struct {
		name     string
		variable string
		value    string
	}{
		{name: "zero database timeout", variable: "DATABASE_CONNECT_TIMEOUT", value: "0s"},
		{name: "negative database timeout", variable: "DATABASE_CONNECT_TIMEOUT", value: "-1s"},
		{name: "zero adapter timeout", variable: "AVITO_ADAPTER_TIMEOUT", value: "0s"},
		{name: "negative adapter timeout", variable: "AVITO_ADAPTER_TIMEOUT", value: "-1s"},
		{name: "zero activation TTL", variable: "TICKET_ACTIVATION_TTL", value: "0s"},
		{name: "negative activation TTL", variable: "TICKET_ACTIVATION_TTL", value: "-1s"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setValidEnvironment(t)
			t.Setenv(test.variable, test.value)

			// when
			_, err := Load()

			// then
			require.Error(t, err)
		})
	}
}

func TestLoad_MissingDependencyAddress_ReturnError(t *testing.T) {
	// given
	tests := []string{"DATABASE_URL", "AVITO_ADAPTER_URL", "RABBITMQ_URL", "SERVICE_AUTH_TOKEN"}

	for _, variable := range tests {
		t.Run(variable, func(t *testing.T) {
			setValidEnvironment(t)
			t.Setenv(variable, "")

			// when
			_, err := Load()

			// then
			require.Error(t, err)
		})
	}
}

func setValidEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("HTTP_HOST", "127.0.0.1")
	t.Setenv("HTTP_PORT", "8090")
	t.Setenv("HTTP_READ_TIMEOUT", "2s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "4s")
	t.Setenv("HTTP_SHUTDOWN_TIMEOUT", "6s")
	t.Setenv("DATABASE_URL", "postgres://tickets:password@postgres/tickets")
	t.Setenv("DATABASE_CONNECT_TIMEOUT", "2s")
	t.Setenv("AVITO_ADAPTER_URL", "http://avito-adapter:8080")
	t.Setenv("AVITO_ADAPTER_TIMEOUT", "3s")
	t.Setenv("RABBITMQ_URL", "amqp://tickets:password@rabbitmq:5672/")
	t.Setenv("SERVICE_AUTH_TOKEN", "queue-to-tickets-secret")
	t.Setenv("TICKET_ACTIVATION_TTL", "12m")
}
