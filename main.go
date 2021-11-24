package main

import (
	"fmt"
	"log"
	"os"
	"sshtunnel/cmd"
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

func main() {
	f, err := os.OpenFile("rssh.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		log.Fatal(err)
	}
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	log.SetOutput(f)
	// modules.InitSession()

	// db, _ := sql.Open("sqlite3", "/opt/sqlite3/logserver.db")
	// modules.ImportAWSInstancesToDB(db, "gwn", "eu-central-1")

	if _, err := os.Stat(".db.ini"); os.IsNotExist(err) {
		fmt.Printf("%s usage:\n%s\n", os.Args[0], instuction())
		return
	}
	cmd.Execute()

	/* a := make([][]string, 0)
	for _, c := range []string{"1", "2", "3", "4"} {
		b := []string{}
		b = append(b, c)
		a = append(a, b)
	}
	fmt.Println(len(a))
	fmt.Println(cap(a)) */
}

func Color(colorString string) func(...interface{}) string {
	sprint := func(args ...interface{}) string {
		return fmt.Sprintf(colorString,
			fmt.Sprint(args...))
	}
	return sprint
}

func instuction() string {
	m := fmt.Sprintf(`
	1. grant all privileges on sshServers.* to ssh@'localhost' identified by 'ssh@gs.com' %s
	2. rssh initdb mysql/sqlite3 %-56s %s
	3. rssh import jph --jh 2.2.2.2  --jp www.gs.com %-36s %s
	4. rssh import file -f instances.csv %-48s %s
	5. rssh import sshkey -k 1.pem -s ssh_user %-42s %s
	6. rssh -p gwn -s %-67s %s
	7. rssh -p gwn -r 1.1.1.1 %-59s %s
	8. rssh -p gwn -r 1.1.1.1 -f files %-50s %s
	`, Purple("# MySQL only, grant ssh user permission"), "", Purple("# initial db"), "", Purple("# import jumper host info (including password/user..)"),
		"", Purple("# import aws/vps instances (additinal attachment)"), "", Purple("# import remote Server ssh key and user"), "", Purple("# list instances stored in db"),
		"", Purple("# remotessh 1.1.1.1 server through jumperHost, default BJ jumperHost used, second JP jumperHost"), "", Purple("# copy files to 1.1.1.1"))

	return m
}
