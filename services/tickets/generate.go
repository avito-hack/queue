package tickets

//go:generate mkdir -p gen/server
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 --config oapi-codegen-cfg.yaml docs/api/openapi.yaml
//go:generate mkdir -p gen/clients/avitoadapter
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 --config oapi-codegen-avito-adapter-client-cfg.yaml ../../schemas/services/avito-adapter/openapi.yaml
//go:generate go run github.com/bdragon300/go-asyncapi/cmd/go-asyncapi@v0.4.0 code --only-sub -t gen/clients/avitoadapterevents ../../schemas/services/avito-adapter/events/events.yaml
//go:generate go run github.com/bdragon300/go-asyncapi/cmd/go-asyncapi@v0.4.0 code -t gen/events docs/events/events.yaml
//go:generate go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
