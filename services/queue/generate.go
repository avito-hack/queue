package queue

//go:generate mkdir -p gen/server
//go:generate mkdir -p gen/clients/avito
//go:generate mkdir -p gen/clients/tickets

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 --config oapi-codegen-server.yaml docs/api/openapi.yaml

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 --config oapi-codegen-avito-client.yaml ../avito-adapter/docs/api/openapi.yaml

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 --config oapi-codegen-tickets-client.yaml ../tickets/docs/api/openapi.yaml
