package database

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3" // init function used only for sqlite3

	_ "github.com/go-sql-driver/mysql" // init function used only for mysql
)

type RSSHConfig struct {
	DBHost string
	DBPort string
	DBUser string
	DBPass string
	DBName string
	DBFile string //`json:"DBFile,omitempty"`
	VPSKey string // vps api key used to get vps instance list
}

type QueryJumperHosts struct {
	JmpHost string
	JmpUser string
	JmpPass string
	JmpPort string
}

const (
	queryTimeout       = 5 * time.Second
	DatabaseName       = "sshServers"
	InstanceTableName  = "instances"
	SSHKeyTableName    = "sshkeys"
	JumpHostsTableName = "jumperHosts"
)

var (
	// SQLite3 database initial sql
	//
	sqlite3InitInstancesTable = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			instance_name TEXT NOT NULL,
			public_ip CHAR(64) NOT NULL,
			private_ip CHAR(64) NOT NULL,
			region TEXT NOT NULL, 
			project CHAR(64) NOT NULL, 
			insert_time  NOT NULL DEFAULT CURRENT_TIMESTAMP, 
			role CHAR(10));
		`, InstanceTableName)

	sqlite3InitSSHKeyTable = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" (
		SSH_USER VARCHAR(10) NOT NULL,
		PRIVATEKEY_CONTENT TEXT NOT NULL,    
		PRIVATEKEY_NAME CHAR(64) NOT NULL,
		PROJECT CHAR(64) NOT NULL, 
		INSERT_TIME  NOT NULL DEFAULT CURRENT_TIMESTAMP, 
		NOTES CHAR(10));
	`, SSHKeyTableName)

	sqlite3InitJumperHostsTable = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" (
		jmphost VARCHAR(25) NOT NULL,
		jmpuser VARCHAR(25) NOT NULL,
		jmppass VARCHAR(200) NOT NULL,
		jmpport VARCHAR(25) NOT NULL DEFAULT 26222,
		latency INT NOT NULL DEFAULT 0,
		insert_time  NOT NULL DEFAULT CURRENT_TIMESTAMP);
	`, JumpHostsTableName)

	// MySQL database initial sql
	mySQLDBInit = fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s DEFAULT character set = 'utf8';", DatabaseName)
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
	mySQLInitInstancesTable = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s ( 
			id BIGINT(20) NOT NULL auto_increment, 
			instance_name VARCHAR(255) NOT NULL , 
			public_ip VARCHAR(25) NOT NULL, 
			private_ip VARCHAR(25) NOT NULL , 
			region VARCHAR(64) NOT NULL , 
			project VARCHAR(64) NOT NULL , 
			insert_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,  
			role VARCHAR(32) NOT NULL, 
			PRIMARY KEY (id)) ENGINE = InnoDB;`, strings.Join([]string{DatabaseName, InstanceTableName}, "."))

	mySQLInitSSHKeyTable = fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s ( 
		id BIGINT(20) NOT NULL auto_increment, 
		ssh_user VARCHAR(10) NOT NULL,
		privateKey_content TEXT NOT NULL, 
		privateKey_name VARCHAR(64) NOT NULL,
		project VARCHAR(64) NOT NULL , 
		insert_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,  
		notes VARCHAR(32) NOT NULL, PRIMARY KEY  (id)) ENGINE = InnoDB;
	`, strings.Join([]string{DatabaseName, SSHKeyTableName}, "."))

	mySQLInitJumperHostsTable = fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s ( 
		id BIGINT(20) NOT NULL auto_increment, 
		jmphost VARCHAR(25) NOT NULL,
		jmpuser VARCHAR(25) NOT NULL,
		jmppass VARCHAR(200) NOT NULL,
		jmpport VARCHAR(25) NOT NULL DEFAULT 26222,
		latency SMALLINT NOT NULL DEFAULT 0,
		insert_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,  		
		PRIMARY KEY  (id)) ENGINE = InnoDB;
	`, strings.Join([]string{DatabaseName, JumpHostsTableName}, "."))

	// default database config file, hidden in `$HOME` directoy
	// DBConFile = ".db.ini"
	DBConFile = path.Join(os.Getenv("HOME"), ".db.ini")
)

func QueryInstancesFromDB(db *sql.DB, project string) []map[string]string {
	sql := fmt.Sprintf("select instance_name,public_ip from %s where project='%s'", InstanceTableName, project)
	rows, err := db.Query(sql)
	if err != nil {
		log.Fatal("query sql failed with error: ", err)
	}
	defer rows.Close()

	// fmt.Printf("%s server list: \n\n", project)
	instances := []map[string]string{}
	for rows.Next() {
		a := map[string]string{}
		var instanceName, publicIP string
		rows.Scan(&instanceName, &publicIP)
		a["Name"] = instanceName
		a["PublicIP"] = publicIP
		instances = append(instances, a)
		// fmt.Println(instanceName, publicIP)
	}
	return instances
}

// only support to query single keyword in sql, like "select jmphost from jumperHosts"
func QueryKeywordFromDB(db *sql.DB, sql string) []string {
	rows, err := db.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var k string
		rows.Scan(&k)
		result = append(result, k)
	}
	return result
}

func GetJumperHostsInfo(db *sql.DB, jmphost string) QueryJumperHosts {
	sql := fmt.Sprintf("select jmphost, jmpuser,jmppass,jmpport from %s where jmphost='%s';", JumpHostsTableName, jmphost)
	rows, err := db.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var result QueryJumperHosts
	for rows.Next() {
		rows.Scan(&result.JmpHost, &result.JmpUser, &result.JmpPass, &result.JmpPort)
	}

	return result
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

	sql := fmt.Sprintf("SELECT EXISTS (%s)", fmt.Sprintf("SELECT PUBLIC_IP FROM %s where PUBLIC_IP='%s'", InstanceTableName, publicIP))

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
	if err := DBExecute(db, sqlite3InitJumperHostsTable); err != nil {
		return err
	}
	return func(sqlite3DBFile string) error {
		data := RSSHConfig{DBFile: sqlite3DBFile}
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
	if err := DBExecute(db, mySQLInitJumperHostsTable); err != nil {
		return err
	}
	// initial mysql connection info and save to local file
	return func(dbhost, dbport, dbuser, dbpass string) error {
		data := RSSHConfig{DBHost: dbhost, DBPort: dbport, DBUser: dbuser, DBPass: dbpass, DBName: "", VPSKey: ""}

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
		conn RSSHConfig
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

func ExportTableTOCSVFile(db *sql.DB, tablename string) {
	// query record data from table
	stmt := fmt.Sprintf("select * from %s", tablename)
	rows, err := db.Query(stmt)
	if err != nil {
		log.Fatal(err)
	}
	columns, err := rows.Columns()
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(columns)
	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(columns))

	for i := range values {
		scanArgs[i] = &values[i]
	}

	totalValues := make([][]string, 0)
	for rows.Next() {

		//Save the contents of each line
		var s []string

		//Add the contents of each line to scanArgs, and also to values
		err = rows.Scan(scanArgs...)
		if err != nil {
			log.Fatal(err.Error())
		}

		for _, v := range values {
			s = append(s, string(v))
			// print(len(s))
		}
		totalValues = append(totalValues, s)
	}
	if len(totalValues) == 0 {
		log.Fatalf("no record found in %s", tablename)
	}
	// write the queried data to local file
	file := tablename + ".csv"
	f, err := os.Create(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	//f.WriteString("\xEF\xBB\xBF")
	w := csv.NewWriter(f)
	for i, row := range totalValues {
		//First write column name + first row of data
		if i == 0 {
			w.Write(columns)
			w.Write(row)
		} else {
			w.Write(row)
		}
	}
	w.Flush()
	fmt.Println("Finished writing to:", file)
}
