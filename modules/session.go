package modules

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"sshtunnel/cipherText"
	"sshtunnel/database"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	// "golang.org/x/crypto/ssh/terminal" deprecated
	"golang.org/x/net/proxy"
	terminal "golang.org/x/term"
)

// AgentInterface ..
type AgentInterface interface{}

// Connect ..
type Connect struct {
	// Client *ssh.Client
	Client *ssh.Client

	// Session
	Session *ssh.Session

	// Session Stdin, Stdout, Stderr...
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	// ProxyDialer
	ProxyDialer proxy.Dialer

	// Connect timeout second.
	ConnectTimeout int

	// SendKeepAliveMax and SendKeepAliveInterval
	SendKeepAliveMax      int
	SendKeepAliveInterval int

	// Session use tty flag.
	TTY bool

	// Forward ssh agent flag.
	ForwardAgent bool

	// ssh-agent interface.
	// agent.Agent or agent.ExtendedAgent
	Agent AgentInterface

	// Forward x11 flag.
	ForwardX11 bool

	/* // shell terminal log flag
	logging bool

	// terminal log add timestamp flag
	logTimestamp bool

	// terminal log path
	logFile string */
}

type cmdOutput struct {
	hostname string
	ip       string
	stde     bytes.Buffer
	stdo     bytes.Buffer
}

const (
	Passcode = "passcode" // used to encrypt/decrypt password or private key
)

// InitSession session
func InitSession(print, fcopy, command bool, proj, role, destPath, rmtHost, rmtPort, rmtUser, rmtPass string, cmdlist, fileList []string) {

	// Ctrl^C  handling in ssh session
	// https://unix.stackexchange.com/questions/102061/ctrl-c-handling-in-ssh-session

	db, err := database.GetDBConnInfo(database.DatabaseName)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	res := ChooseJumperHost(db)
	if proj != "" {
		if print {
			result := database.QueryInstancesFromDB(db, proj, "")
			sortStrings := make([]string, 0, len(result))
			for _, v := range result {
				sortStrings = append(sortStrings, v["Name"])
				// for k := range v {
				// 	sortStrings = append(sortStrings, k)
				// }

			}
			sort.Strings(sortStrings)
			fmt.Printf("\n%s Server [Total Count: %d] List: \n\n", strings.ToUpper(proj), len(result))
			fmt.Printf("%-45s| %-15s| %-15s |%15s |\n", "Name", "PublicIP", "InstanceType", "InstanceID")
			for _, k := range sortStrings {
				for _, m := range result {
					if m["Name"] == k {
						// fmt.Println(m)
						fmt.Println(strings.Repeat("-", 102))
						fmt.Printf("%-45s| %-15s| %-15s |%15s |\n", k, m["PublicIP"], m["InstanceType"], m["InstanceID"])
					}
					// if _, ok := m[k]; ok {
					// 	// fmt.Println(k, len(m[k]))
					// 	fmt.Println(strings.Repeat("-", 102))
					// 	fmt.Printf("%-45s| %-15s| %-15s |%15s |\n", k, m[k][0], m[k][1], m[k][2])
					// }

				}
			}
			return
		}

		var privateKey string
		sshinfo := GetSSHKey(db, proj, Passcode)
		switch filepath.Ext(sshinfo.PrivateKeyName) {
		case ".pass":
			rmtUser = sshinfo.SSHUser
			// privateKey = ""
			rmtPass = sshinfo.PrivateKeyContent
		case ".pem":
			rmtUser = sshinfo.SSHUser
			// rmtPass = ""
			privateKey = sshinfo.PrivateKeyContent
		default:
			fmt.Printf("no ssh_key/password record found for %s (PROJECT %s) in DB `sshkeys`, use your input pass instead\n", rmtHost, proj)
			// return
		}

		// if rmtHost != "" {
		// 	makeProxyHost(
		// 		res.JmpHost,
		// 		res.JmpUser,
		// 		res.JmpPass,
		// 		res.JmpPort,
		// 		rmtHost,
		// 		rmtPort,
		// 		rmtUser,
		// 		rmtPass,
		// 		privateKey,
		// 		proj,
		// 		destPath,
		// 		role,
		// 		command,
		// 		fcopy,
		// 		cmdlist,
		// 		fileList)
		// }
		makeProxyHost(
			res.JmpHost,
			res.JmpUser,
			res.JmpPass,
			res.JmpKey,
			res.JmpPort,
			rmtHost,
			rmtPort,
			rmtUser,
			rmtPass,
			privateKey,
			proj,
			destPath,
			role,
			command,
			fcopy,
			cmdlist,
			fileList)

	} else {
		fmt.Printf("lack of `-project` parameter, will connect to Jump server %s directly\n", rmtHost)

		if rmtHost == "" {
			fmt.Printf("no `rmtHost` parameter specified, connect to Jump server %s directly\n", res.JmpHost)
			makeDirectSSH(res.JmpHost, res.JmpUser, res.JmpPass, res.JmpKey, res.JmpPort, proj, destPath, fcopy, fileList)
			return
		}
		// makeDirectSSH(res.JmpHost, res.JmpUser, res.JmpPass, res.JmpPort, proj, destPath, fcopy, fileList)
		makeDirectSSH(rmtHost, rmtUser, rmtPass, "", rmtPort, proj, destPath, fcopy, fileList)
	}
	// network connection quality check
	// networkdetect.LatencyTest("52.83.235.118", 26222)
}

// get the low latency jump host used to connect the backend server
func ChooseJumperHost(db *sql.DB) database.QueryJumperHosts {
	var (
		jpass, jkey []byte
		err         error
	)

	sql := "select jmphost from jumperHosts where latency=(select MIN(latency) from jumperHosts)"
	minLatencyJmpHosts := database.QueryKeywordFromDB(db, sql)
	if len(minLatencyJmpHosts) == 0 {
		log.Fatal("not JumperHost found in db")
	}
	log.Println("min latency jump host", minLatencyJmpHosts)
	res := database.GetJumperHostsInfo(db, minLatencyJmpHosts[0])
	if res.JmpPass != "" {
		jpass, err = cipherText.DecryptData(res.JmpPass, Passcode)
		if err != nil {
			log.Fatal(err)
		}
	}

	if res.JmpKey != "" {
		jkey, err = cipherText.DecryptData(res.JmpKey, Passcode)
		if err != nil {
			log.Fatal(err)
		}
	}

	res.JmpKey = string(jkey)
	res.JmpPass = string(jpass)
	return res
}

func linuxShell(session *ssh.Session) {
	// works well on linux system
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	terminalModes := ssh.TerminalModes{
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
	if err = session.RequestPty(xterm, hight, width, terminalModes); err != nil {
		log.Fatalf("request pty error %s", err)
	}

	// resize terminal window size dynamicly
	winch := syscall.Signal(0x1c)
	signalchan := make(chan os.Signal, 1)
	signal.Notify(signalchan, winch)
	go func() {
		for {
			s := <-signalchan
			switch s {
			case winch:
				fd := int(os.Stdout.Fd())
				width, hight, _ = terminal.GetSize(fd)
				session.WindowChange(hight, width)
			}
		}
	}()

	if err = session.Shell(); err != nil {
		log.Fatalf("start shell error %s", err)
	}
	if err = session.Wait(); err != nil {
		log.Fatalf("return error: %s", err)
	}
}
func makeDirectSSH(jmpHost, jmpUser, jmpPass, jmpkey, jmpPort, proj, destPath string, fcopy bool, fileList []string) {
	// make client
	jumpHost := net.JoinHostPort(jmpHost, jmpPort)
	sshConfig := InitSSHClientConfig(jmpUser, jmpPass, jmpkey, proj, 20)
	client, err := ssh.Dial("tcp", jumpHost, &sshConfig)

	if err != nil {
		log.Fatalf("dial %s failed with error %s", jmpHost, err.Error())
	}
	defer client.Close()
	// c := Connect{Client: client}
	// if filename == "" {
	if !fcopy {
		// make session
		session, err := client.NewSession()
		if err != nil {
			log.Fatal("new session failed with error: ", err)
		}
		defer session.Close()
		// c.xShell(session)
		linuxShell(session)

	} else {
		localCopy(client, destPath, fileList)
	}
}

func makeProxyHost(jmpHost, jmpUser, jmpPass, jmpkey, jmpPort, rmtHost, rmtPort, rmtUser, rmtPass, privateKey, proj, destPath, role string, command, fcopy bool, cmdList, fileList []string) {
	jumpHost := net.JoinHostPort(jmpHost, jmpPort)

	proxyConn := Connect{}
	err := proxyConn.createClient(jumpHost, jmpUser, jmpPass, jmpkey, proj)
	if err != nil {
		log.Fatal("failed to connect jumper host with error: ", err)
	}
	// target connnect
	targetConn := Connect{
		ProxyDialer: proxyConn.Client,
	}

	// remoteHost := net.JoinHostPort(rmtHost, rmtPort)

	// err = targetConn.createClient(remoteHost, rmtUser, rmtPass, privateKey, proj)
	// if err != nil {
	// 	fmt.Printf("failed to connect remote Host %s with error %s\n", remoteHost, err)
	// 	os.Exit(1)
	// }

	if role != "" {
		db, err := database.GetDBConnInfo(database.DatabaseName)
		if err != nil {
			log.Fatal(err)
		}
		hostinfo := database.QueryInstancesFromDB(db, proj, role)
		ch := make(chan cmdOutput, len(hostinfo))
		// ch := make(chan cmdOutput)

		if command {
			// var wg sync.WaitGroup
			cmd := strings.Join(cmdList, " ")

			for _, host := range hostinfo {
				// wg.Add(1)
				go func(hostname, ip, c string) {
					var res cmdOutput
					// defer wg.Done()

					err = targetConn.createClient(net.JoinHostPort(ip, rmtPort), rmtUser, rmtPass, privateKey, proj)
					if err != nil {
						fmt.Printf("failed to connect remote Host %s with error %s\n", ip, err)
						os.Exit(1)
					}
					session, err := targetConn.Client.NewSession()
					if err != nil {
						fmt.Printf("launch new session for %s failed with error %s in cmd excution\n", ip, err)
						os.Exit(1)

					}
					defer session.Close()
					res.hostname = hostname
					res.ip = ip
					session.Stdout = &res.stdo
					session.Stderr = &res.stde
					if err = session.Run(cmd); err != nil {
						switch e := err.(type) {
						case *ssh.ExitError:
							log.Printf("run cmd (%s) failed on %s with error %s\n", cmd, ip, e.Waitmsg)
						case *ssh.ExitMissingError:
							log.Printf("run cmd (%s) failed on %s with error %s\n", cmd, ip, e.Error())
						default:
							log.Printf("run cmd (%s) failed on %s with error %s (%s)", cmd, ip, e, res.stde.String())
						}
					}
					ch <- res
				}(host["Name"], host["PublicIP"], cmd)
			}
			for range hostinfo {
				res := <-ch
				fmt.Printf("{\n\tHost: %s \n", Green(strings.Join([]string{res.hostname, res.ip}, "-")))
				if res.stde.String() == "" {
					fmt.Printf("\tStatus: %s\n", Green("SUCCESS"))
					fmt.Printf("\tResult: %20s", res.stdo.String())
				} else {
					fmt.Printf("\tStatus: %s\n", Red("FAILED"))
					fmt.Printf("\tResult: %s\n", res.stde.String())
				}
				fmt.Printf("}\n\n")
			}
			/*
				// another way to implement that reading data from zero length channnel `ch`
					go func() { wg.Wait(); close(ch) }()
					for res := range ch {
						fmt.Printf("{\n\tHost: %s \n", Green(strings.Join([]string{res.hostname, res.ip}, "-")))
						if res.stde.String() == "" {
							fmt.Printf("\tStatus: %s\n", Green("SUCCESS"))
							fmt.Printf("\tResult: %20s", res.stdo.String())
						} else {
							fmt.Printf("\tStatus: %s\n", Red("FAILED"))
							fmt.Printf("\tResult: %s\n", res.stde.String())
						}
						fmt.Printf("}\n")
					}
			*/
		} else if fcopy {
			for _, host := range hostinfo {
				err = targetConn.createClient(net.JoinHostPort(host["PublicIP"], rmtPort), rmtUser, rmtPass, privateKey, proj)
				if err != nil {
					fmt.Printf("failed to connect remote Host %s with error %s\n", host["PublicIP"], err)
					os.Exit(1)
				}
				localCopy(targetConn.Client, destPath, fileList)
			}
		} else {
			fmt.Printf("Add %s to execute cmd or %s to copy file after the currently present args\n\n", Green("-c"), Green("-f"))
			if len(hostinfo) == 0 {
				sql := fmt.Sprintf("select distinct(role) from %s", database.InstanceTableName)

				fmt.Printf("role %s not found, refer to available role list: %s\n", Green(role), database.QueryKeywordFromDB(db, sql))
				os.Exit(1)
			}
			fmt.Printf("project %s's %s instances:\n", Green(proj), Green(role))
			for _, host := range hostinfo {
				fmt.Printf("\t%s\t%s\n", host["Name"], host["PublicIP"])
			}
		}

	} else {
		if rmtHost == "" {
			fmt.Printf("%s no remoteHost provided to be connected, add %s/%s to get/connnect instance (list)\n", Red("Error:"), Green("-s"), Green("-r"))
			os.Exit(1)
		}
		remoteHost := net.JoinHostPort(rmtHost, rmtPort)
		err = targetConn.createClient(remoteHost, rmtUser, rmtPass, privateKey, proj)
		if err != nil {
			fmt.Printf("failed to connect remote Host %s with error %s\n", remoteHost, err)
			os.Exit(1)
		}

		if fcopy {
			localCopy(targetConn.Client, destPath, fileList)
		} else {
			session, err := targetConn.Client.NewSession()
			if err != nil {
				log.Fatal("new remote host session failed with error: ", err)
			}
			defer session.Close()
			targetConn.xShell(session)
			// linuxShell(session)
		}
	}

}

func (c *Connect) createClient(host, user, password, privateKey, proj string) (err error) {

	// Create new ssh.ClientConfig{}

	timeout := 20
	if c.ConnectTimeout > 0 {
		timeout = c.ConnectTimeout
	}

	config := InitSSHClientConfig(user, password, privateKey, proj, timeout)

	// check Dialer
	if c.ProxyDialer == nil {
		c.ProxyDialer = proxy.Direct
	}

	// Dial to host:port
	netConn, err := c.ProxyDialer.Dial("tcp", host)
	if err != nil {
		return err
	}

	// Create new ssh connect
	sshCon, channel, req, err := ssh.NewClientConn(netConn, host, &config)
	if err != nil {
		return err
	}

	// Create *ssh.Client
	c.Client = ssh.NewClient(sshCon, channel, req)
	return
}

// Shell .
func (c *Connect) xShell(session *ssh.Session) (err error) {
	// Input terminal Make raw
	fd := int(os.Stdin.Fd())
	state, err := terminal.MakeRaw(fd)
	if err != nil {
		return
	}
	defer terminal.Restore(fd, state)

	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	err = RequestTty(session)
	if err != nil {
		return err
	}

	// Start shell
	err = session.Shell()
	if err != nil {
		return
	}

	// keep alive packet
	go c.SendKeepAlive(session)

	err = session.Wait()
	if err != nil {
		return
	}

	return
}

// RequestTty .
func RequestTty(session *ssh.Session) (err error) {
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.ECHOCTL:       1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	// Get terminal window size
	fd := int(os.Stdin.Fd())
	width, hight, err := terminal.GetSize(fd)
	if err != nil {
		return err
	}

	// TODO(blacknon): 環境変数から取得する方式だと、Windowsでうまく動作するか不明なので確認して対処する
	if err = session.RequestPty("xterm", hight, width, modes); err != nil {
		session.Close()
		return
	}

	// Terminal resize goroutine.
	winch := syscall.Signal(0x1c)
	signalchan := make(chan os.Signal, 1)
	signal.Notify(signalchan, winch)
	go func() {
		for {
			s := <-signalchan
			switch s {
			case winch:
				fd := int(os.Stdout.Fd())
				width, hight, _ = terminal.GetSize(fd)
				session.WindowChange(hight, width)
			}
		}
	}()

	return
}

// SendKeepAlive .
func (c *Connect) SendKeepAlive(session *ssh.Session) {
	// keep alive interval (default 30 sec)
	interval := 30
	if c.SendKeepAliveInterval > 0 {
		interval = c.SendKeepAliveInterval
	}

	// keep alive max (default 5)
	max := 5
	if c.SendKeepAliveMax > 0 {
		max = c.SendKeepAliveMax
	}

	// keep alive counter
	i := 0
	for {
		// Send keep alive packet
		_, err := session.SendRequest("keepalive", true, nil)
		// _, _, err := c.Client.SendRequest("keepalive", true, nil)
		if err == nil {
			i = 0
		} else {
			i++
		}

		// check counter
		if max <= i {
			session.Close()
			return
		}

		// sleep
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
