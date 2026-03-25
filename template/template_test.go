package template

import (
	"strings"
	"testing"
)

func TestRenderTextAndProjectHelpers(t *testing.T) {
	buf, err := RenderText("{{.ProjectName}}|{{.KebabName}}|{{.GRPCVersionPackageName}}", Project{Name: "hello_world"})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(buf); got != "HelloWorld|hello-world|hello_worldv1" {
		t.Fatalf("unexpected render output: %s", got)
	}
}

func TestProjectHelpers(t *testing.T) {
	p := Project{Name: "hello_world", Driver: "mariadb"}
	if p.DriverName() != "GetMariaDB" {
		t.Fatalf("unexpected driver name: %s", p.DriverName())
	}
	if p.PackageName() != "helloworld" {
		t.Fatalf("unexpected package name: %s", p.PackageName())
	}
	if p.ProjectName() != "HelloWorld" || p.ModelName() != "HelloWorld" {
		t.Fatalf("unexpected project/model name: %s %s", p.ProjectName(), p.ModelName())
	}
	if p.RouteName() != "hello-world" || p.KebabName() != "hello-world" {
		t.Fatalf("unexpected route/kebab: %s %s", p.RouteName(), p.KebabName())
	}
	if p.TagsName() != "hello_world" || p.TableName() != "hello_world" {
		t.Fatalf("unexpected tags/table: %s %s", p.TagsName(), p.TableName())
	}
	if p.GRPCPackageName() != "hello_world" || p.GRPCVersionPackageName() != "hello_worldv1" {
		t.Fatalf("unexpected grpc package names: %s %s", p.GRPCPackageName(), p.GRPCVersionPackageName())
	}
	p.ThirdParty = "Core Service"
	if p.ThirdPartyName() != "core_service" {
		t.Fatalf("unexpected thirdparty name: %s", p.ThirdPartyName())
	}
}

func TestRenderTextSupportsTemplateFuncs(t *testing.T) {
	buf, err := RenderText(`{{contains "abcdef" "bc"}}|{{sub 5 3}}|{{index (split "a,b" ",") 1}}`, Any{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(buf)) != "true|2|b" {
		t.Fatalf("unexpected helper output: %s", string(buf))
	}
}

func TestRenderTextSupportsHasField(t *testing.T) {
	buf, err := RenderText(`{{hasField .Fields "CreatedBy"}}|{{hasField .Fields "UpdatedBy"}}`, Project{
		Fields: []Field{{Name: "CreatedBy"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(buf)); got != "true|false" {
		t.Fatalf("unexpected hasField output: %s", got)
	}
}
