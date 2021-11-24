package main

import (
	"log"
	"os"
	"sshtunnel/cmd"
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
