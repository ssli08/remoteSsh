package modules

import (
	"fmt"
	"os"
)

func CheckExist() {
	if _, err := os.Stat("/path/to/whatever"); os.IsNotExist(err) {
		// path/to/whatever does not exist
		fmt.Println("not exist")
	}

	if _, err := os.Stat("/path/to/whatever"); !os.IsNotExist(err) {
		// path/to/whatever exists
		fmt.Println("exist")
	}
}
