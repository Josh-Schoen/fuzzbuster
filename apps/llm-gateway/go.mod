module github.com/fuzzbuster/fuzzbuster/apps/llm-gateway

go 1.23

require (
	github.com/fuzzbuster/fuzzbuster/pkg v0.0.0
	github.com/go-chi/chi/v5 v5.1.0
)

replace github.com/fuzzbuster/fuzzbuster/pkg => ../../pkg
