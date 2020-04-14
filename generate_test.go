package cmap
import (
	"fmt"
	"strings"
	"testing"
)
func TestGenerateTypeSyncMap(t *testing.T) {
	fmt.Println(GenerateTypeSyncMap("User", map[string]string{
		"${package_name}": "model",
	}))
}

func TestReplace(t *testing.T) {
	var r = `
	   ${package_name}
	   ${model}
	   ${Model}
	   ${MODEL}
	`
	fmt.Println(strings.Replace(r, "${package_name}", "model", -1))

	fmt.Println(replace(r, map[string]string{
		"${package_name}": "model",
		"${Model}":"User",
		"${model}": "user",
		"${MODEL}": "USER",
	}))
}