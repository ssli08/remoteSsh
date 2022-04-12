package accessToken

import (
	"database/sql"
	"fmt"
	"log"
	"sshtunnel/database"
	"strings"

	"gopkg.in/ini.v1"
)

// get RABC policy from local file
func PolicyFromLocalFile(filename, rolename string) []string {
	cfg, err := ini.Load(filename)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := cfg.GetSection("permission"); err != nil {
		log.Fatal(err)
	}
	if cfg.Section("permission").HasKey(rolename) {
		cmdlist := cfg.Section("permission").Key(rolename).String()
		log.Printf("cmd list for role %s is %s", rolename, strings.Split(cmdlist, ","))
		return strings.Split(cmdlist, ",")
	}
	return nil
}

// get RABC policy from db
func policyFromDB(db *sql.DB) []string {
	sql := fmt.Sprintf("SELECT * FROM %s WHERE rolename=%s", "", "rolename")
	actions := database.QueryKeywordFromDB(db, sql)
	log.Println("got action list from db ", actions)
	return actions
}
