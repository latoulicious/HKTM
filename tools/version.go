package tools

import (
	"fmt"

	"github.com/latoulicious/HKTM/internal/version"
)

func Versioning() {
	info := version.Get()
	fmt.Println(info.String())
}
