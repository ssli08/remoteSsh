package modules

import "os"

func check() {
	if _, err := os.Stat("/path/to/whatever"); os.IsNotExist(err) {
		// path/to/whatever does not exist
	}

	if _, err := os.Stat("/path/to/whatever"); !os.IsNotExist(err) {
		// path/to/whatever exists
	}
}
