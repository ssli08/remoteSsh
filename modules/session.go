package modules

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
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

const (
	Passcode = "passcode" // used to encrypt/decrypt password or private key
)

// InitSession session
func InitSession(print, fcopy, directly bool, proj, destPath, rmtHost, rmtPort, rmtUser, rmtPass string, fileList []string) {

	// Ctrl^C  handling in ssh session
	// https://unix.stackexchange.com/questions/102061/ctrl-c-handling-in-ssh-session

	db, err := database.GetDBConnInfo(database.DatabaseName)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if proj != "" && print {
		result := database.QueryInstancesFromDB(db, proj)
		fmt.Printf("\n%s Server [Total Count: %d] List: \n\n", strings.ToUpper(proj), len(result))

		fmt.Println(strings.Repeat("-", 95))
		fmt.Printf("%-45s| %-15s| %10s |%10s|\n", "Name", "PublicIP", "InstanceType", "InstanceID")
		for _, i := range result {
			fmt.Println(strings.Repeat("-", 95))
			fmt.Printf("%-45s| %-15s| %10s |%10s|\n", i["Name"], i["PublicIP"], i["InstanceType"], i["InstanceID"])
		}
		fmt.Println()
		return
	}

	res := ChooseJumperHost(db)
	if rmtHost != "" {
		if proj != "" {
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
			makeProxyHost(res.JmpHost, res.JmpUser, res.JmpPass, res.JmpPort, rmtHost, rmtPort, rmtUser, rmtPass, privateKey, proj, destPath, fcopy, fileList)
		} else {
			fmt.Printf("lack of `-project` parameter, will connect to Jump server %s directly\n", rmtHost)
			// makeDirectSSH(res.JmpHost, res.JmpUser, res.JmpPass, res.JmpPort, proj, destPath, fcopy, fileList)
			makeDirectSSH(rmtHost, rmtUser, rmtPass, rmtPort, proj, destPath, fcopy, fileList)
		}
	} else {
		fmt.Printf("no `rmtHost` parameter specified, connect to Jump server %s directly\n", res.JmpHost)
		makeDirectSSH(res.JmpHost, res.JmpUser, res.JmpPass, res.JmpPort, proj, destPath, fcopy, fileList)
	}

	// network connection quality check
	// networkdetect.LatencyTest("52.83.235.118", 26222)
}

func ChooseJumperHost(db *sql.DB) database.QueryJumperHosts {
	sql := "select jmphost from jumperHosts where latency=(select MIN(latency) from jumperHosts)"
	minLatencyJmpHosts := database.QueryKeywordFromDB(db, sql)
	if len(minLatencyJmpHosts) == 0 {
		log.Fatal("not JumperHost found in db")
	}
	log.Println("min latency jump host", minLatencyJmpHosts)
	res := database.GetJumperHostsInfo(db, minLatencyJmpHosts[0])
	pass, err := cipherText.DecryptData(res.JmpPass, Passcode)
	if err != nil {
		log.Fatal(err)
	}
	res.JmpPass = string(pass)
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
func makeDirectSSH(jmpHost, jmpUser, jmpPass, jmpPort, proj, destPath string, fcopy bool, fileList []string) {
	// make client
	jumpHost := net.JoinHostPort(jmpHost, jmpPort)
	sshConfig := InitSSHClientConfig(jmpUser, jmpPass, "", proj, 20)
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

func makeProxyHost(jmpHost, jmpUser, jmpPass, jmpPort, rmtHost, rmtPort, rmtUser, rmtPass, privateKey, proj, destPath string, fcopy bool, fileList []string) {
	jumpHost := net.JoinHostPort(jmpHost, jmpPort)

	proxyConn := Connect{}
	err := proxyConn.createClient(jumpHost, jmpUser, jmpPass, privateKey, proj)
	if err != nil {
		log.Fatal("failed to connect jumper host with error: ", err)
	}
	// target connnect
	targetConn := Connect{
		ProxyDialer: proxyConn.Client,
	}

	remoteHost := net.JoinHostPort(rmtHost, rmtPort)

	err = targetConn.createClient(remoteHost, rmtUser, rmtPass, privateKey, proj)
	if err != nil {
		// log.Fatal("remote host connect failed with error: ", err)
		fmt.Printf("failed to connect remote Host %s with error %s\n", remoteHost, err)
		os.Exit(1)
	}

	// if filename == "" {
	if !fcopy {
		session, err := targetConn.Client.NewSession()
		if err != nil {
			log.Fatal("new remote host session failed with error: ", err)
		}
		defer session.Close()
		targetConn.xShell(session)
		// linuxShell(session)
	} else {
		// localCopy(targetConn.Client, filename)
		localCopy(targetConn.Client, destPath, fileList)
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
