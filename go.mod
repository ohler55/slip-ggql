module github.com/ohler55/slip-ggql

go 1.21

require (
	github.com/ohler55/ojg v1.20.2
	github.com/ohler55/slip v0.5.0
	github.com/uhn/ggql v1.2.14
)

replace github.com/ohler55/slip => ../slip

require golang.org/x/text v0.14.0 // indirect
