package modules

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/term"
)

const (
	InfoColor    = "\033[1;34m%s\033[0m"
	NoticeColor  = "\033[1;36m%s\033[0m"
	WarningColor = "\033[1;33m%s\033[0m"
	ErrorColor   = "\033[1;31m%s\033[0m"
	DebugColor   = "\033[0;36m%s\033[0m"
)

var (
	Black   = Color("\033[1;30m%s\033[0m")
	Red     = Color("\033[1;31m%s\033[0m")
	Green   = Color("\033[1;32m%s\033[0m")
	Yellow  = Color("\033[1;33m%s\033[0m")
	Purple  = Color("\033[1;34m%s\033[0m")
	Magenta = Color("\033[1;35m%s\033[0m")
	Teal    = Color("\033[1;36m%s\033[0m")
	White   = Color("\033[1;37m%s\033[0m")
)

func Color(colorString string) func(...interface{}) string {
	sprint := func(args ...interface{}) string {
		return fmt.Sprintf(colorString,
			fmt.Sprint(args...))
	}
	return sprint
}

// print json format content
func JsonOutput(data interface{}) (string, error) {
	res, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return "", fmt.Errorf("json data output error %s", err)
	}
	return string(res), nil
}
func GetInputPassword() (string, error) {
	// reader := bufio.NewReader(os.Stdin)
	// fmt.Print("Enter Username: ")
	// username, _ := reader.ReadString('\n')

	fmt.Print("Enter Password: ")
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println()
	// return strings.TrimSpace(username), strings.TrimSpace(password)
	return strings.TrimSpace(string(bytePassword)), nil
}
