echo "clean all"
rm -rf /var/log/snmp_exporter/*
rm -rf snmp_exporter
echo "git pull"
git pull
echo "go build all"
go build -a -o snmp_exporter main.go collector.go  config.go 
echo "running snmp_exporter"
./snmp_exporter --config.dir config
