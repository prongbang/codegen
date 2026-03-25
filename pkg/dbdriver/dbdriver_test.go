package dbdriver

import "testing"

func TestDriverConstants(t *testing.T) {
	if DriverMysql != "mysql" {
		t.Fatalf("unexpected mysql driver: %s", DriverMysql)
	}
	if DriverMariaDb != "mariadb" {
		t.Fatalf("unexpected mariadb driver: %s", DriverMariaDb)
	}
	if DriverPostgres != "postgres" {
		t.Fatalf("unexpected postgres driver: %s", DriverPostgres)
	}
	if DriverMongoDb != "mongodb" {
		t.Fatalf("unexpected mongodb driver: %s", DriverMongoDb)
	}
}
