module github.com/fuzzbuster/fuzzbuster/apps/orchestrator

go 1.23

require (
	github.com/fuzzbuster/fuzzbuster/pkg v0.0.0
	github.com/hibiken/asynq v0.25.1
	github.com/jackc/pgx/v5 v5.7.1
	github.com/mmcdole/gofeed v1.3.0
)

replace github.com/fuzzbuster/fuzzbuster/pkg => ../../pkg
