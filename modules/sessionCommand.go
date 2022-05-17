package modules

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	terminal "golang.org/x/term"
)

// another implement of filtering cmd input by user
// refer to https://mritd.com/2018/11/09/go-interactive-shell/
func SessionCommand(session *ssh.Session) error {
	// fmt.Println("Requesting Pseudo Terminal")
	modes := ssh.TerminalModes{
		ssh.ECHO:          1, // 0 disable echoing, 1 enable echoing
		ssh.ECHOCTL:       1,
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	hight := 55
	width := 211
	fd := int(os.Stdin.Fd())
	if terminal.IsTerminal(int(fd)) {
		width, hight, _ = terminal.GetSize(int(fd))
	}

	orgState, err := terminal.MakeRaw(fd)
	if err != nil {
		log.Fatal(err)
	}
	defer terminal.Restore(fd, orgState)
	xterm := os.Getenv("TERM")
	if xterm == "" {
		xterm = "xterm-256color"
	}
	if err := session.RequestPty("xterm", hight, width, modes); err != nil {
		fmt.Println("Unable to request Pseudo Terminal")
		fmt.Println("Error : ", err.Error())
		return err
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return err
	}

	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				fmt.Println(err)
				return
			}
			if n > 0 {
				// fmt.Printf("%s", buf[:n])
				if bytes.Contains(buf[:n], []byte("r")) {
					if _, err := stdin.Write([]byte("no permission\n")); err != nil {
						fmt.Println(err)
						return
					}
					continue
				}

				_, err = stdin.Write(buf[:n])
				if err != nil {
					fmt.Println(err)
					return
				}
			}
		}

		// scanner := bufio.NewScanner(os.Stdin)

		// scanner.Scan()
		// content := scanner.Bytes()
		// // content := scanner.Text()
		// if strings.Contains(string(content), "rm") {
		// 	stdin.Write([]byte("echo no permission"))
		// 	return
		// }
		// stdin.Write(content)

	}()

	if err := session.Shell(); err != nil {
		log.Fatal("exit err:", err)
	}
	if err := session.Wait(); err != nil {
		log.Fatal("wait err: ", err)
	}
	return nil
}

// for filtering cmd input by user
func SessionCmd(session *ssh.Session) {
	var (
		stdin          io.WriteCloser
		stdout, stderr io.Reader
		err            error
	)

	defer session.Close()

	stdin, err = session.StdinPipe()
	if err != nil {
		fmt.Println(err.Error())
	}

	stdout, err = session.StdoutPipe()
	if err != nil {
		fmt.Println(err.Error())
	}

	stderr, err = session.StderrPipe()
	if err != nil {
		fmt.Println(err.Error())
	}

	wr := make(chan []byte, 1024)

	go func() {
		for {
			c := <-wr
			fmt.Println(string(c))
			_, err := stdin.Write(c)
			if err != nil {
				fmt.Println("input error: ", err.Error())
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stdout)
		for {
			if tkn := scanner.Scan(); tkn {
				rcv := scanner.Bytes()
				raw := make([]byte, len(rcv))
				copy(raw, rcv)

				fmt.Println("output:", string(raw))
			} else {
				if scanner.Err() != nil {
					fmt.Println(scanner.Err())
				} else {
					fmt.Println("io.EOF")
				}
				return
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)

		for scanner.Scan() {
			fmt.Println("std error: ", scanner.Text())
		}
	}()

	if err = session.Shell(); err != nil {
		log.Fatal("start shell failed ", err)
	}
	// if err = session.Wait(); err != nil {
	// 	log.Fatal("session wait error ", err)
	// }

	for {
		// fmt.Println("$")

		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		text := scanner.Text()
		if strings.Contains(text, "rm") {
			text = "echo no permission"
		}

		wr <- []byte(fmt.Sprintf("%s\n", text))
	}
}
