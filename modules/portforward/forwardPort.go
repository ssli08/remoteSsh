package portforward

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

var (
	remoteAddr string
	remotePort string
	localAddr  string
	jumperHost string

	sshUser string
	sshPass string
	sshPort string
)

func init() {
	flag.StringVar(&localAddr, "lAddr", "127.0.0.1:7000", "local listening address for ssh connection")
	flag.StringVar(&remoteAddr, "rAddr", "", "remote server ip for ssh connection")
	flag.StringVar(&remotePort, "rport", "", "port for remote host ssh connection")
	flag.StringVar(&jumperHost, "jHost", "52.83.235.118", "jumper host for ssh connection")
	flag.StringVar(&sshUser, "jUser", "ec2-user", "user for ssh connection")
	flag.StringVar(&sshPass, "sshPass", "", "password for jumper host ssh connection")
	flag.StringVar(&sshPort, "sshPort", "26222", "port for jump host ssh connection")
	flag.Parse()
}

// PortForward .
func PortForward() {
	// Connection settings
	if remoteAddr == "" || sshPass == "" {
		log.Fatal("error occured, lack remoteAddr or sshPass")
	}
	// Build SSH client configuration
	cfg, err := makeSSHConfig(sshUser, sshPass)
	if err != nil {
		log.Fatalln(err)
	}

	// Establish connection with SSH server
	conn, err := ssh.Dial("tcp", net.JoinHostPort(jumperHost, sshPort), cfg)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	// Establish connection with remote server
	// remote, err := conn.Dial("tcp", remoteAddr)
	// if err != nil {
	// 	log.Fatalf("connect remote %s failed with err: %s .", remoteAddr, err)
	// }

	// Start local server to forward traffic to remote connection
	local, err := net.Listen("tcp", localAddr)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("listening on ", localAddr)
	defer local.Close()

	// Handle incoming connections
	for {
		remote, err := conn.Dial("tcp", net.JoinHostPort(remoteAddr, remotePort))
		if err != nil {
			log.Fatalf("connect remote %s failed with err: %s .", net.JoinHostPort(remoteAddr, remotePort), err)
		}

		client, err := local.Accept()
		if err != nil {
			log.Fatalln(err)
		}

		go handleClient(client, remote)
	}
}

// Get default location of a private key
func privateKeyPath() string {
	return os.Getenv("HOME") + "/.ssh/id_rsa"
}

// Get private key for ssh authentication
func parsePrivateKey(keyPath string) (ssh.Signer, error) {
	buff, _ := ioutil.ReadFile(keyPath)
	return ssh.ParsePrivateKey(buff)
}

// Get ssh client config for our connection
// SSH config will use 2 authentication strategies: by key and by password
func makeSSHConfig(user, password string) (*ssh.ClientConfig, error) {
	key, err := parsePrivateKey(privateKeyPath())
	if err != nil {
		// fmt.Println(err)
		return nil, err
	}

	config := ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	return &config, nil
}

// Handle local client connections and tunnel data to the remote serverq
// Will use io.Copy - http://golang.org/pkg/io/#Copy
func handleClient(client, remote net.Conn) {

	defer client.Close()
	log.Println("get connection from ", client.RemoteAddr())
	chDone := make(chan bool)

	// Start remote -> local data transfer
	go func() {
		_, err := io.Copy(client, remote)
		if err != nil {
			log.Println("error while copy remote->local:", err)
		}

		b := make([]byte, 1)
		if _, err = client.Read(b); err == io.EOF {
			log.Printf("connection from %s closed. ", client.RemoteAddr())
		}
		// chDone <- true
	}()

	// Start local -> remote data transfer
	go func() {
		_, err := io.Copy(remote, client)
		if err != nil {
			log.Println(err)
		}
		// chDone <- true
	}()

	<-chDone
}

func isConnAlive(conn net.Conn) net.Conn {
	b := make([]byte, 1)
	_, err := conn.Read(b)

	if err == io.EOF {
		log.Printf("%s detected closed.", conn.LocalAddr().String())
		return nil
	}
	log.Printf("%v is alive", conn.LocalAddr().String())
	// log.Printf("set timeout for %s", conn.LocalAddr().String())
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	return conn
}
