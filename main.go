package main

import (
	"log"
	"sshtunnel/modules"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	modules.InitSession()
	// parseKeys()
	// portforward.PortForward()
	// modules.GetInstanceInfo()
	// modules.ImportVPSInstanceInfoToDB()

	// modules.ReadXLSX("gwn.xlsx")
	// db, _ := sql.Open("sqlite3", "/opt/sqlite3/logserver.db")
	// modules.ImportAWSInstancesToDB(db, "gwn", "eu-central-1")

	// cmd.Execute()
	// db, _ := sql.Open("sqlite3", "/opt/sqlite3/logserver.db")
	// modules.ImportSSHKey(db, "gdms.pem", "rmtssh")
	// modules.ExportSSHKey(db, "gdms", "rmtssh")
	// cipherText.CT()

}

// func networkTest() {
// 	modules.ForSingnalUsage()
// 	for _, ip := range []string{"52.83.235.118", "13.115.186.176"} {
// 		networkdetect.LatencyTest(ip, 26222)
// 	}
// }
