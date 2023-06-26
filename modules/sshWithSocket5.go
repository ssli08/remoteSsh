package modules

import (
	"fmt"
	"log"

	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"
)

func mainT() {
	// ssh remote server with socket5 proxy support

	sshConfig := &ssh.ClientConfig{
		User: "user",
		// Auth: .... fill out with keys etc as normal
	}

	client, err := proxiedSSHClient("127.0.0.1:8000", "127.0.0.1:22", sshConfig)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(client)
	// get a session etc...
}

func proxiedSSHClient(proxyAddress, sshServerAddress string, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
	dialer, err := proxy.SOCKS5("tcp", proxyAddress, nil, proxy.Direct)
	if err != nil {
		return nil, err
	}

	conn, err := dialer.Dial("tcp", sshServerAddress)
	if err != nil {
		return nil, err
	}

	c, chans, reqs, err := ssh.NewClientConn(conn, sshServerAddress, sshConfig)
	if err != nil {
		return nil, err
	}

	return ssh.NewClient(c, chans, reqs), nil
}
