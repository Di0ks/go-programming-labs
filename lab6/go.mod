module github.com/Di0ks/go-programming-labs/lab6

go 1.26.5

replace github.com/Di0ks/go-programming-labs/utils => ../utils

require (
	github.com/Di0ks/go-programming-labs/lab3/mathutils v0.0.0-00010101000000-000000000000
	github.com/Di0ks/go-programming-labs/lab6/pool v0.0.0-00010101000000-000000000000
	github.com/Di0ks/go-programming-labs/utils v0.0.0-00010101000000-000000000000
)

replace github.com/Di0ks/go-programming-labs/lab3/mathutils => ../lab3/mathutils

replace github.com/Di0ks/go-programming-labs/lab6/pool => ./pool
