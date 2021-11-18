package modules

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path"
	"sshtunnel/cipherText"
	"sshtunnel/database"
	"strings"
)

// import ssh key to specified db and use `passphrase` as key to encrypt ssh key content
// encrypt program:
// [string --> encrypted --> base64 encode --> db]
func ImportSSHKey(db *sql.DB, keyFile, passphrase string) {
	buf, err := os.ReadFile(keyFile)
	if err != nil {
		log.Fatal(err)
	}

	// encrypted key
	econtent, err := cipherText.EncryptData(buf, passphrase)
	if err != nil {
		log.Fatal(err)
	}
	c := base64.StdEncoding.EncodeToString(econtent)
	sql := fmt.Sprintf("INSERT INTO sshkeys (project, privateKey_name, privateKey_content) values ('%s','%s', '%s')", strings.TrimSuffix(keyFile, ".pem"), path.Base(keyFile), c)
	if err := database.DBExecute(db, sql); err != nil {
		log.Fatal(err)
	}
}

// return sshkey map
// decrypted program:
// [encyptedString --> base64 decode --> decrypted --> map[string]string]
func GetSSHKey(db *sql.DB, project, passphrase string) map[string]string {
	sql := fmt.Sprintf("SELECT privateKey_name, privateKey_content FROM sshkey WHERE project='%s'", project)
	rows, err := db.Query(sql)
	if err != nil {
		log.Fatal("query sql failed with error: ", err)
	}
	defer rows.Close()
	sshKey := map[string]string{}

	for rows.Next() {
		var privateKeyName, privateKeyContent string
		rows.Scan(&privateKeyName, &privateKeyContent)
		c, err := base64.StdEncoding.DecodeString(privateKeyContent)
		if err != nil {
			log.Fatal(err)
		}
		key, err := cipherText.DecryptData(c, passphrase)
		if err != nil {
			log.Fatal(err)
		}
		sshKey[privateKeyName] = string(key)
	}
	// fmt.Println(sshKey)
	return sshKey
}
