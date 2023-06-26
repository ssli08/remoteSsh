package modules

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/blacknon/go-sshlib"
	"golang.org/x/crypto/ssh"
)

var (
	remoteHost string
	remotePort string
	remotePass string

	jumperHost string
	jumperPort string

	user string
	pass string
)

func init() {

	flag.StringVar(&remoteHost, "remoteHost", "", "remote server ip for ssh connection")
	flag.StringVar(&remotePort, "remotePort", "26222", "port of remote host for ssh connetion")
	flag.StringVar(&remotePass, "remotePass", "ec2@gs.com", "password of remote host for ssh connetion")

	flag.StringVar(&jumperHost, "jumperHost", "52.83.235.118", "jumper host for ssh connection")
	flag.StringVar(&jumperPort, "jumperPort", "26222", "port of jumper host for ssh connection")

	flag.StringVar(&user, "sshUser", "ec2-user", "user for ssh connection")
	flag.StringVar(&pass, "sshPass", "", "password for ssh connection")
	// flag.StringVar(&sshPort, sshPort, "26222", "port for ssh connection")
	flag.Parse()
}

var (
	// Proxy ssh server
	// jumperHost = "proxy.com"
	// jumperPort      = "22"
	// user      = "user"
	// pass  = "password"

	// Target ssh server
	// remoteHost     = "target.com"
	// remotePort     = "22"
	// user     = "user"
	// pass = "password"

	termlog = "./test_termlog"
)

func mainC() {

	// ==========
	// proxy connect
	// ==========

	// Create proxy sshlib.Connect
	proxyCon := &sshlib.Connect{}

	// Create proxy ssh.AuthMethod
	proxyAuthMethod := sshlib.CreateAuthMethodPassword(pass)

	// Connect proxy server
	err := proxyCon.CreateClient(jumperHost, jumperPort, user, []ssh.AuthMethod{proxyAuthMethod})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// ==========
	// target connect
	// ==========

	// Create target sshlib.Connect
	targetCon := &sshlib.Connect{
		ProxyDialer: proxyCon.Client,
	}

	// Create target ssh.AuthMethod
	targetAuthMethod := sshlib.CreateAuthMethodPassword(remotePass)

	// Connect target server
	err = targetCon.CreateClient(remoteHost, remotePort, user, []ssh.AuthMethod{targetAuthMethod})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Set terminal log
	targetCon.SetLog(termlog, false)

	// Start ssh shell
	targetCon.Shell(targetCon.Session)
}

func generateConfig() *ssh.ClientConfig {
	key, err := ioutil.ReadFile(func() string { return os.Getenv("HOME") + "/.ssh/id_rsa" }())
	if err != nil {
		log.Fatal(err)
	}
	sign, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatal(err)
	}

	cfg := ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(sign),
			ssh.Password(pass),
		},
	}
	return &cfg
}
