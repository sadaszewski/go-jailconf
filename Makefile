all: jailconf

jailconf: jailconf.peg.go jailconf.go jailconf_raw.go
	go build jailconf.go jailconf_raw.go jailconf.peg.go 

jailconf.peg.go: jailconf.peg
	peg jailconf.peg
