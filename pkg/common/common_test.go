package common

import "testing"

func TestCommonHelpers(t *testing.T) {
	if got := Map([]string{"a", "b"}, func(item string) string { return item + "!" }); got != "a!b!" {
		t.Fatalf("unexpected map output: %s", got)
	}
	if got := UpperCamelName("hello_world"); got != "HelloWorld" {
		t.Fatalf("unexpected upper camel: %s", got)
	}
	if got := LowerCamelName("hello_world_test"); got != "helloWorldTest" {
		t.Fatalf("unexpected lower camel: %s", got)
	}
	if got := ToLower("Hello-World"); got != "helloworld" {
		t.Fatalf("unexpected to lower: %s", got)
	}
	if got := Abbrev("helloWorld-service_name"); got != "hwsn" {
		t.Fatalf("unexpected abbrev: %s", got)
	}
}
