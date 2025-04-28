module github.com/ohler55/slip-ggql

go 1.24

require (
	github.com/ohler55/ojg v1.26.4
	github.com/ohler55/slip v1.0.1
	github.com/uhn/ggql v1.2.14
)

replace github.com/ohler55/slip => ../slip

require (
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/term v0.21.0 // indirect
	golang.org/x/text v0.19.0 // indirect
)
