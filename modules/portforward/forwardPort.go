package portforward

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"time"

	"golang.org/x/crypto/ssh"
)

// PortForward .
func PortForward(sshUser, sshPass, sshPort, sshHost, localAddr, rmtAddr, rmtPort string) {
	// Build SSH client configuration
	cfg, err := makeSSHConfig(sshUser, sshPass)
	if err != nil {
		log.Fatalln(err)
	}

	// Establish connection with SSH server
	conn, err := ssh.Dial("tcp", net.JoinHostPort(sshHost, sshPort), cfg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer conn.Close()

	// Start local server to forward traffic to remote connection
	local, err := net.Listen("tcp", localAddr)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("listening on ", localAddr)
	defer local.Close()

	// Handle incoming connections

	for {
		remote, err := conn.Dial("tcp", net.JoinHostPort(rmtAddr, rmtPort))
		if err != nil {
			log.Fatalf("connect remote %s failed with err: %s .", net.JoinHostPort(rmtAddr, rmtPort), err)
		}
		client, err := local.Accept()
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("get connection from %s\n", client.RemoteAddr().String())

		go handleClient(client, remote)
	}
}

// Get ssh client config for our connection
// SSH config will use 2 authentication strategies: by key and by password
func makeSSHConfig(user, password string) (*ssh.ClientConfig, error) {
	var Auth []ssh.AuthMethod

	if password == "" {

		buf, err := os.ReadFile(path.Join(os.Getenv("HOME") + ".ssh/id_rsa"))
		if err != nil {
			return nil, err
		}
		key, err := ssh.ParsePrivateKey(buf)
		if err != nil {
			// fmt.Println(err)
			return nil, err
		}
		Auth = append(Auth, ssh.PublicKeys(key))
	} else {
		Auth = append(Auth, ssh.Password(password))
	}

	config := ssh.ClientConfig{
		User:            user,
		Auth:            Auth,
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
