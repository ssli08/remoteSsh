package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sshtunnel/cmd"
	"sshtunnel/database"
	"sshtunnel/modules"
	"syscall"
)

var Purple = modules.Purple

func mainA() {

}
func main() {

	if runtime.GOOS == "windows" {
		log.Fatal("not support windows yet!!!")
	}

	file := "/tmp/rssh.log"
	syscall.Umask(000)
	f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		log.Fatal(err)
	}
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	log.SetOutput(f)
	// modules.InitSession()

	// db, _ := sql.Open("sqlite3", "/opt/sqlite3/logserver.db")
	// modules.ImportAWSInstancesToDB(db, "gwn", "eu-central-1")
	if _, err := os.Stat(database.DBConFile); os.IsNotExist(err) && len(os.Args) < 2 {
		fmt.Printf("%s usage:\n%s\n", os.Args[0], instuction())
		return
	}

	cmd.Execute()
	// modules.MultiProgressBarPresentation(500, 10)

}

func instuction() string {
	m := fmt.Sprintf(`
	1. grant all privileges on sshServers.* to ssh@'localhost' identified by 'ssh@gs.com' %s
	2. rssh initdb -h for mysql/sqlite3 initialization %-56s %s
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
