module github.com/fuzzbuster/fuzzbuster/apps/moderation

go 1.23

require (
	github.com/fuzzbuster/fuzzbuster/pkg v0.0.0
	github.com/jackc/pgx/v5 v5.7.1
)

replace github.com/fuzzbuster/fuzzbuster/pkg => ../../pkg
