package tickets

//go:generate mkdir -p gen/server
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 --config oapi-codegen-cfg.yaml api/openapi.yaml
//go:generate mkdir -p gen/clients/avitoadapter
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 --config oapi-codegen-avito-adapter-client-cfg.yaml ../../schemas/services/avito-adapter/openapi.yaml
