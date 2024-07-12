
buf:
	buf generate

sql:
	sqddl generate -dialect mysql -prefix protos  -dest ./internal/db/models.go -output-dir ./internal/db/migrations
