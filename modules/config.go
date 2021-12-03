package modules

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"runtime"

	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts" // module for parsing `known_hosts` file
)

var (
	knownHostFile string = path.Join(os.Getenv("HOME"), ".ssh/known_hosts")
	keyErr        *knownhosts.KeyError
)

func InitSSHClientConfig(user, password, privateKey, proj string, timeout int) ssh.ClientConfig {
	var auth []ssh.AuthMethod
	if password == "" {
		// db, err := database.GetDBConnInfo(database.DatabaseName)
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// defer db.Close()

		// keyname,sshUser, sshKey := GetSSHKey(db, proj, Passcode)
		// if user == "" {
		// 	user = sshUser
		// }

		buf := []byte(privateKey)
		key, err := ssh.ParsePrivateKey(buf)
		if err != nil {
			// log.Fatalf("parse key file failed with error %s, Input password instead", err)
			fmt.Printf("parse key file failed with error %s, Input password instead\n", err)
			os.Exit(1)
		}
		auth = append(auth, ssh.PublicKeys(key))
	} else {
		// if user == "" {
		// 	// log.Fatalf("no ssh user provided, %s `--ru ssh_user` ", os.Args[0])
		// 	fmt.Printf("no ssh user provided, %s `--ru ssh_user`\n", os.Args[0])
		// 	os.Exit(1)
		// }
		auth = append(auth, ssh.Password(password))
	}

	// hostkey check
	var hostKeyCheck ssh.HostKeyCallback
	if runtime.GOOS != "windows" {
		hostKeyCheck = ssh.HostKeyCallback(func(host string, remote net.Addr, pubKey ssh.PublicKey) error {
			nhkc := newHostKeyCallback()
			err := nhkc(host, remote, pubKey)
			if errors.As(err, &keyErr) && len(keyErr.Want) > 0 {
				log.Printf("WARNING: %v is not a key of %s, either a MiTM attack or %s has reconfigured the host pub key.", hostKeyString(pubKey), host, host)
				return keyErr
			} else if errors.As(err, &keyErr) && len(keyErr.Want) == 0 {
				log.Printf("WARNING: %s is not trusted, adding this key: %q to known_hosts file.", host, hostKeyString(pubKey))
				addToKnownHostsFile(host, remote, pubKey)
				return nil
			}
			// log.Printf("Pub key %s exists for %s.", hostKeyString(pubKey), host)
			log.Printf("Pub key exists for %s.", host)
			return nil

		})
	} else {
		hostKeyCheck = ssh.InsecureIgnoreHostKey() // windows system without sshHostKey
	}
	config := ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: hostKeyCheck,
		Timeout:         time.Duration(timeout) * time.Second,
	}
	// fmt.Printf("%+v\n", config)
	return config
}

// pase multiple ssh_keys in local file, eg: `~/.ssh/id_rsa`
func ParseSSHKeys() []byte {
	// NOT used in remoteSsh
	// auto_generation.pem
	homePath := os.Getenv("HOME")

	buf, err := ioutil.ReadFile(homePath + "/.ssh/id_rsa")
	if err != nil {
		fmt.Println(err)
	}
	var sshKeys []byte
	for _, n := range strings.SplitAfter(string(buf), "-----END RSA PRIVATE KEY-----") {
		s := []byte(n)
		sshKeys = append(sshKeys, s...) // Attention, three dots notation
	}

	return sshKeys
}

// create human-readable SSH-key strings defined in known_hosts file
func hostKeyString(k ssh.PublicKey) string {
	return k.Type() + " " + base64.StdEncoding.EncodeToString(k.Marshal()) // e.g. "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY...."
}

// create ssh.HostKeyCallback
func newHostKeyCallback() ssh.HostKeyCallback {
	hostKeyCallBack, err := knownhosts.New(knownHostFile)
	if err != nil {
		log.Fatal("an error occured during creating HostKeyCallback: ", err)
	}
	return hostKeyCallBack
}

// add remote host host publicKey to knonw_hosts file
func addToKnownHostsFile(host string, remote net.Addr, pubKey ssh.PublicKey) error {
	f, _ := os.OpenFile(knownHostFile, os.O_APPEND|os.O_WRONLY, 0644)
	defer f.Close()

	knownHostLine := knownhosts.Normalize(remote.String())
	hostItem := knownhosts.Line([]string{knownHostLine}, pubKey)

	// hash host value and write to `known_hosts` file
	hostItemSlice := strings.Split(hostItem, " ")
	hashHost := knownhosts.HashHostname(hostItemSlice[0])
	hostItemSlice[0] = hashHost
	_, err := f.WriteString(fmt.Sprintf("%s\n", strings.Join(hostItemSlice, " ")))
	return err
}
