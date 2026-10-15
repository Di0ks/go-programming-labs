module github.com/Di0ks/go-programming-labs/lab10/restapi

go 1.26.5

replace github.com/Di0ks/go-programming-labs/utils => ../utils

replace github.com/Di0ks/go-programming-labs/lab7/httpserv => ../../lab7/httpserv

require (
	github.com/Di0ks/go-programming-labs/lab7/httpserv v0.0.0-00010101000000-000000000000
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/jackc/pgx/v5 v5.11.0
	golang.org/x/crypto v0.57.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)
