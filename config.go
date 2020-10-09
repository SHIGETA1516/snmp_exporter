package main

type Target struct {
	Host string
	Port uint16
	Oids []struct {
		Oid        string
		Metricname string
	}
}

type Config struct {
	Global struct {
		Address    string
		Interval   string
		TimeOut    int
		Oidtimeout int
		Oidretries int
	}
	Targets []Target
}
