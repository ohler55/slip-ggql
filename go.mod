module github.com/ohler55/slip-ggql

go 1.25

require (
	github.com/ohler55/ojg v1.27.0
	github.com/ohler55/slip v1.3.0
	github.com/uhn/ggql v1.2.14
)

replace github.com/ohler55/slip => ../slip

require (
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/term v0.34.0 // indirect
	golang.org/x/text v0.28.0 // indirect
)
