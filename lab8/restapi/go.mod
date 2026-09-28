module github.com/Di0ks/go-programming-labs/lab8/restapi

go 1.27.1

replace github.com/Di0ks/go-programming-labs/lab7/httpserv => ../../lab7/httpserv

require (
	github.com/Di0ks/go-programming-labs/lab7/httpserv v0.0.0-00010101000000-000000000000
	github.com/jackc/pgx/v5 v5.11.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)
