module github.com/Di0ks/go-programming-labs/lab7

go 1.26.5

replace github.com/Di0ks/go-programming-labs/utils => ../utils

require (
	github.com/Di0ks/go-programming-labs/lab7/httpserv v0.0.0-00010101000000-000000000000
	github.com/Di0ks/go-programming-labs/utils v0.0.0-00010101000000-000000000000
	golang.org/x/net v0.59.0
)

replace github.com/Di0ks/go-programming-labs/lab7/httpserv => ./httpserv
