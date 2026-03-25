package arch_test

import (
	"fmt"
	"testing"

	"github.com/prongbang/codegen/pkg/arch"
)

func TestIsDarwinArm64(t *testing.T) {
	a := arch.New()
	fmt.Println(a.IsDarwinArm64())
}
