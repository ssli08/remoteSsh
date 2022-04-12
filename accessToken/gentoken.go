package accessToken

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"sshtunnel/database"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	WarningStmt = "Warning: you need an auhtorization to Execute this cmd"
)

// func toekn(project string) {
// 	base64.StdEncoding.EncodeToString([]byte(""))
// }

// GenerateToken returns a unique token based on the provided project string
func GenerateToken(project, user, role string) string {
	db, _ := database.GetDBConnInfo(database.DatabaseName)
	hash, err := bcrypt.GenerateFromPassword([]byte(strings.Join([]string{project, user}, "-")), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println("Hash to store:", string(hash))
	hasher := md5.New()
	hasher.Write(hash)
	token := hex.EncodeToString(hasher.Sum(nil))
	ip, mac := GetLocalIPMAC()
	ipm := strings.Join([]string{ip, mac}, "-")
	sql := fmt.Sprintf("INSERT INTO %s (username,rolename,project,ipmac,token) VALUES('%s','%s','%s','%s','%s');",
		database.AccessTokenTableName, user, role, project, ipm, token)
	database.DBExecute(db, sql)
	return token
	// return base64.StdEncoding.EncodeToString(hash)
}

// get local ip and mac address which executed this script
func GetLocalIPMAC() (string, string) {
	var (
		ch              = make(chan string)
		ifname, ip, mac string
		err             error
	)

	iplist := []string{"114.114.114.114:53", "8.8.8.8:53"}
	for _, ipa := range iplist {
		go func(ip string) {
			conn, err := net.Dial("udp", ip)
			if err != nil {
				log.Fatal(err)
			}
			defer conn.Close()

			localAddr := conn.LocalAddr().(*net.UDPAddr)
			// fmt.Println(localAddr.IP.String())
			ch <- localAddr.String()

		}(ipa)
	}

	if ip, _, err = net.SplitHostPort(<-ch); err != nil {
		log.Fatal(err)
	}

	interfaces, _ := net.Interfaces()
	for _, interf := range interfaces {

		if addrs, err := interf.Addrs(); err == nil {
			for _, addr := range addrs {
				// only interested in the name with current IP address
				if strings.Contains(addr.String(), ip) {
					log.Println("got network Interface Name ", interf.Name)
					ifname = interf.Name
				}
			}
		}
	}
	if netInterface, err := net.InterfaceByName(ifname); err == nil {
		mac = netInterface.HardwareAddr.String()
	}

	return ip, mac
}
