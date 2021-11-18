package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3" // init function used only for sqlite3

	_ "github.com/go-sql-driver/mysql" // init function used only for mysql
)

type mySQLConnInfo struct {
	DBHost string
	DBPort string
	DBUser string
	DBPass string
	DBName string
	DBFile string //`json:"DBFile,omitempty"`
}

const (
	queryTimeout = 5 * time.Second

	sqlite3InitInstancesTable = `CREATE TABLE IF NOT EXISTS "instances" (
			INSTANCE_NAME TEXT NOT NULL,
			PUBLIC_IP CHAR(64) NOT NULL,
			PRIVATE_IP CHAR(64) NOT NULL,
			REGION TEXT NOT NULL, PROJECT CHAR(64) NOT NULL, INSERT_TIME  NOT NULL DEFAULT CURRENT_TIMESTAMP, ROLE CHAR(10));
		`
	sqlite3InitSSHKeyTable = `CREATE TABLE IF NOT EXISTS "sshkeys" (
		PRIVATEKEY_CONTENT TEXT NOT NULL,    
		PRIVATEKEY_NAME CHAR(64) NOT NULL,
		PROJECT CHAR(64) NOT NULL, INSERT_TIME  NOT NULL DEFAULT CURRENT_TIMESTAMP, NOTES CHAR(10));
	`
	mySQLDBInit = "CREATE DATABASE IF NOT EXISTS sshServers DEFAULT character set = 'utf8';"
	// original `create table` temp
	/*
		CREATE TABLE `sshServers`.`myproject` (
			`id` BIGINT(20) NOT NULL ,
			`instance_name` VARCHAR(255) NOT NULL ,
			`public_ip` VARCHAR(25) NOT NULL,
			`private_ip` VARCHAR(25) NOT NULL ,
			`region` VARCHAR(64) NOT NULL ,
			`project` VARCHAR(64) NOT NULL ,
			`insert_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			`role` VARCHAR(32) NOT NULL,
			PRIMARY KEY (`id`)) ENGINE = InnoDB;
	*/
	mySQLInitInstancesTable = `CREATE TABLE IF NOT EXISTS sshServers.instances ( 
			id BIGINT(20) NOT NULL auto_increment, 
			instance_name VARCHAR(255) NOT NULL , 
			public_ip VARCHAR(25) NOT NULL, 
			private_ip VARCHAR(25) NOT NULL , 
			region VARCHAR(64) NOT NULL , 
			project VARCHAR(64) NOT NULL , 
			insert_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,  
			role VARCHAR(32) NOT NULL, 
			PRIMARY KEY (id)) ENGINE = InnoDB;`

	mySQLInitSSHKeyTable = `
	CREATE TABLE IF NOT EXISTS sshServers.sshkeys ( 
		id BIGINT(20) NOT NULL auto_increment, 
		privateKey_content TEXT NOT NULL, 
		privateKey_name VARCHAR(64) NOT NULL,
		project VARCHAR(64) NOT NULL , 
		insert_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,  
		notes VARCHAR(32) NOT NULL, PRIMARY KEY  (id)) ENGINE = InnoDB;
	`
)

var (
	DBConFile = ".db.ini"
)

func QueryDB(project string) {
	db, err := sql.Open("sqlite3", "/opt/sqlite3/logserver.db")
	if err != nil {
		log.Fatal("access db error: ", err)
	}
	defer db.Close()

	sql := fmt.Sprintf("select instance_name,public_ip from myproject where project='%s'", project)
	rows, err := db.Query(sql)
	if err != nil {
		log.Fatal("query sql failed with error: ", err)
	}
	defer rows.Close()
	// p := []map[string]string{}
	fmt.Printf("%s server list: \n\n", project)
	for rows.Next() {
		var instanceName string
		var publicIP string
		rows.Scan(&instanceName, &publicIP)
		fmt.Println(instanceName, publicIP)
		// p = append(p, map[string]string{instanceName: publicIP})
	}
}

func DBExecute(db *sql.DB, sql string) error {
	// connection pool setup
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Minute * 1)

	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	log.Println("execute sql ", sql)
	result, err := db.ExecContext(ctx, sql)
	if err != nil {
		return err
	}

	row, err := result.RowsAffected()
	if err != nil {
		return err
	}

	// if !strings.Contains(sql, "CREATE") {
	// 	if row != 1 {
	// 		// log.Fatalf("expected to affect 1 row, affected %d", row)
	// 		return fmt.Errorf("expected to affect 1 row, affected %d", row)
	// 	}
	// }
	log.Printf("expected to affect 1 row, affected %d", row)
	return nil
}

// check if `PUBLIC_IP` record exist in DB
func IsRecordExist(db *sql.DB, publicIP string) bool {
	var exists bool

	sql := fmt.Sprintf("SELECT EXISTS (%s)", fmt.Sprintf("SELECT PUBLIC_IP FROM instances where PUBLIC_IP='%s'", publicIP))

	row := db.QueryRow(sql)
	if err := row.Scan(&exists); err != nil {
		log.Println(err)
	}
	return exists
}

func SQLite3Init(sqlite3DBFile string) error {
	db, err := sql.Open("sqlite3", sqlite3DBFile)
	if err != nil {
		return fmt.Errorf("access sqlite3 error %s", err)
	}
	defer db.Close()
	if err := DBExecute(db, sqlite3InitInstancesTable); err != nil {
		return err
	}
	if err := DBExecute(db, sqlite3InitSSHKeyTable); err != nil {
		return err
	}
	return func(sqlite3DBFile string) error {
		data := mySQLConnInfo{DBFile: sqlite3DBFile}
		f, err := os.OpenFile(DBConFile, os.O_WRONLY, 0755)
		if os.IsNotExist(err) {
			f, _ = os.OpenFile(DBConFile, os.O_CREATE|os.O_WRONLY, 0755)
		}
		e := json.NewEncoder(f)
		if err := e.Encode(data); err != nil {
			return err
		}
		return nil
	}(sqlite3DBFile)
}

func MySQLConnInit(dbHost, dbport, user, pass string) error {
	// [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/", user, pass, dbHost, dbport))
	if err != nil {
		return err
	}
	// test if db connection is alive
	/*
		 ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
		err = db.PingContext(ctx)
		cancel()
	*/

	defer db.Close()
	if err := DBExecute(db, mySQLDBInit); err != nil {
		return err
	}
	if err := DBExecute(db, mySQLInitInstancesTable); err != nil {
		return err
	}
	if err := DBExecute(db, mySQLInitSSHKeyTable); err != nil {
		return err
	}

	// initial mysql connection info and save to local file
	return func(dbhost, dbport, dbuser, dbpass string) error {
		data := mySQLConnInfo{DBHost: dbhost, DBPort: dbport, DBUser: dbuser, DBPass: dbpass}

		f, err := os.OpenFile(DBConFile, os.O_WRONLY, 0755)
		if os.IsNotExist(err) {
			f, _ = os.OpenFile(DBConFile, os.O_CREATE|os.O_WRONLY, 0755)
		}
		e := json.NewEncoder(f)
		if err := e.Encode(data); err != nil {
			return err
		}
		return nil
	}(dbHost, dbport, user, pass)

}

// get db info from `DBConFile` and return `*sql.DB`
func GetDBConnInfo(dbname string) (*sql.DB, error) {
	var (
		conn mySQLConnInfo
		db   *sql.DB
	)

	f, err := os.Open(DBConFile)
	if err != nil {
		return nil, err
	}
	d := json.NewDecoder(f)
	if err := d.Decode(&conn); err != nil {
		return nil, err
	}
	if conn.DBFile == "" {
		if db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", conn.DBUser, conn.DBPass, conn.DBHost, conn.DBPort, dbname)); err != nil {
			return nil, fmt.Errorf("init mysql driver failed with error %s", err)
		}
	} else {
		if db, err = sql.Open("sqlite3", conn.DBFile); err != nil {
			return nil, fmt.Errorf("init sqlite3 driver failed with error %s", err)
		}
	}
	return db, nil
}
