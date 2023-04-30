module github.com/ohler55/slip-ggql

go 1.20

require (
	github.com/ohler55/ojg v1.18.5
	github.com/ohler55/slip v0.5.0
	github.com/uhn/ggql v1.2.14
)

replace github.com/ohler55/slip => ../slip

require golang.org/x/text v0.4.0 // indirect
