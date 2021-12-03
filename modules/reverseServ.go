package modules

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

func ReverseServAndExCmd() {
	cmd := os.Args[1]
	hosts := os.Args[2:]

	result := make(chan string, len(hosts))
	timeout := time.After(10 * time.Second)

	user := os.Getenv("USER")
	if user == "kk" {
		user = "root"
	}
	var auth []ssh.AuthMethod
	auth = append(auth, ssh.Password("gdms@cloud"))

	cfg := ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	for _, host := range hosts {
		go func(hostname, port string) {
			result <- excecuteCmd(cmd, hostname, port, &cfg)
		}(host, "22")
	}
	for i := 0; i < len(hosts); i++ {
		select {
		case res := <-result:
			fmt.Println(res)
		case <-timeout:
			fmt.Println("TimeOut!")
			return

		}
	}
	// reverseServ(os.Args[2:][0], "22", &cfg)
}

func excecuteCmd(cmd, host, port string, config *ssh.ClientConfig) string {
	// fmt.Println(config)
	conn, err := ssh.Dial("tcp", net.JoinHostPort(host, port), config)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	var ebuf, obuf bytes.Buffer
	session.Stderr = &ebuf
	session.Stdout = &obuf

	if err = session.Run(cmd); err != nil {
		log.Fatal(err)
	}
	if ebuf.String() != "" {
		return fmt.Sprintf("%s -> \nerr: %s", host, ebuf.String())
	}
	// if err = session.Start(cmd); err != nil {
	// 	log.Fatal(err)
	// }
	// if err = session.Wait(); err != nil {
	// 	log.Fatal(err)
	// }
	return fmt.Sprintf("%s -> \n%s", host, obuf.String())

}

func reverseServ(host, port string, config *ssh.ClientConfig) {
	conn, err := ssh.Dial("tcp", net.JoinHostPort(host, port), config)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// l, err := conn.Listen("tcp", net.JoinHostPort(host, "8000"))
	l, err := conn.ListenTCP(&net.TCPAddr{IP: net.ParseIP(host), Port: 8000})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(l.Addr())
	defer l.Close()

	if err := http.Serve(l, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(rw, "hello world!\n")
	})); err != nil {
		log.Fatal(err)
	}
}
