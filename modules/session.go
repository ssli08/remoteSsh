package modules

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sshtunnel/database"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/net/proxy"
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

	// shell terminal log flag
	logging bool

	// terminal log add timestamp flag
	logTimestamp bool

	// terminal log path
	logFile string
}

const (
	// jmpHost = "13.115.186.176"
	jmpUser = "ec2-user"
	// jmpPass = "ec2@gs.com"
	jmpPort = "26222"
	proto   = "tcp"
)

var (
	// terminalModes = ssh.TerminalModes{
	// 	ssh.ECHO: 1, // 0 disable echoing, 1 enable echoing
	// 	// ssh.ECHOCTL:       0,
	// 	ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
	// 	ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	// }
	// jmpHost string
	// jmpUser string
	// jmpPass string
	// jmpPort string
	jph     bool
	rmtHost string
	rmtUser string
	rmtPass string
	rmtPort string

	proj string
	// proto string

	print bool

	fcopy bool //file copy
)

/* func init() {

	flag.StringVar(&proj, "p", "", "project server to connect and show server list, options: gwn|gdms|ipvt")

	// flag.StringVar(&jmpHost, "jh", "52.83.235.118", "jumper host|13.115.186.176")
	// flag.StringVar(&jmpUser, "ju", "ec2-user", "jumper host user for ssh connection")
	// flag.StringVar(&jmpPass, "jp", "ec2gs.com", "jumper host password for ssh connection")
	// flag.StringVar(&jmpPort, "jport", "26222", "jumper ssh port for connection")
	flag.BoolVar(&jph, "jp", false, "if true login through `jp`, else trough bj")
	flag.StringVar(&rmtHost, "rh", "", "remote host")
	flag.StringVar(&rmtUser, "ru", "ec2-user", "remote user")
	flag.StringVar(&rmtPass, "rp", "", "remote password")
	flag.StringVar(&rmtPort, "rport", "26222", "remote server ssh port for connection")

	// flag.StringVar(&proto, "prtc", "tcp", "default connect protocol")

	// flag.StringVar(&filename, "file", "", "filename to copy")
	flag.BoolVar(&fcopy, "f", false, "file copy")

	flag.BoolVar(&print, "print", false, "print server list in conrresponding project")

	flag.Parse()
} */
func init() {
	flag.StringVar(&proj, "p", "gdms", "project server to connect and show server list")
	flag.BoolVar(&jph, "jp", false, "if true login through `jp`, else trough bj")
	flag.StringVar(&rmtHost, "rh", "34.217.185.246", "remote host")
	flag.StringVar(&rmtUser, "ru", "yxxu", "remote user")
	flag.StringVar(&rmtPass, "rp", "yxxu@gs.com", "remote password")
	flag.StringVar(&rmtPort, "rport", "26222", "remote server ssh port for connection")
	flag.BoolVar(&fcopy, "f", false, "file copy")

	flag.BoolVar(&print, "print", false, "print server list in conrresponding project")

	flag.Parse()
}

// InitSession session
func InitSession() {
	// if jmpPass == "" { //|| rmtHost == ""  || rmtPass == "" {
	// projct := []string{"gwn", "gdms", "ipvt"}
	// cmd := fmt.Sprintf("select instance_name,public_ip from myproject where project=%s", proj)
	// out := exec.Command(cmd)

	// Ctrl^C  handling in ssh session
	// https://unix.stackexchange.com/questions/102061/ctrl-c-handling-in-ssh-session

	var (
		jmpHost string
		jmpPass string
	)

	if jph {
		jmpHost = "13.115.186.176"
		jmpPass = "ec2@gs.com"
	} else {
		jmpHost = "52.83.235.118"
		jmpPass = "4475@gs.com"
	}

	if proj != "" && print {
		database.QueryDB(proj)
		fmt.Println()
		return
	}

	if rmtHost != "" {
		if proj != "" {
			makeProxyHost(jmpHost, jmpPass)
		} else {
			fmt.Println(fmt.Errorf("lack of `-project` parameter, will connect to Jump server directly"))
		}
	} else {
		fmt.Println(fmt.Errorf("No `rmtHost` parameter specified, connect to Jump server directly"))
		makeDirectSSH(jmpHost, jmpPass)
	}

	// network connection quality check
	// networkdetect.LatencyTest("52.83.235.118", 26222)
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

	if err = session.RequestPty("xterm", hight, width, terminalModes); err != nil {
		log.Fatalf("request pty error %s", err)
	}
	if err = session.Shell(); err != nil {
		log.Fatalf("start shell error %s", err)
	}
	if err = session.Wait(); err != nil {
		log.Fatalf("return error: %s", err)
	}
}
func makeDirectSSH(jmpHost, jmpPass string) {
	// make client
	jumpHost := net.JoinHostPort(jmpHost, jmpPort)
	sshConfig := makeClientConfig(jmpUser, jmpPass, 20)
	client, err := ssh.Dial(proto, jumpHost, &sshConfig)

	if err != nil {
		log.Fatalf("dial %s failed with error %s", jmpHost, err.Error())
	}
	defer client.Close()

	// if filename == "" {
	if !fcopy {
		// make session
		session, err := client.NewSession()
		if err != nil {
			log.Fatal("new session failed with error: ", err)
		}
		defer session.Close()

		linuxShell(session)
	} else {

		localCopy(client, jmpHost, flag.Args())
	}
}

func makeProxyHost(jmpHost, jmpPass string) {
	jumpHost := net.JoinHostPort(jmpHost, jmpPort)

	proxyConn := Connect{}
	err := proxyConn.createClient(jumpHost, jmpUser, jmpPass)
	if err != nil {
		log.Fatal("jumper host connect failed with error: ", err)
	}
	// target connnect
	targetConn := Connect{
		ProxyDialer: proxyConn.Client,
	}

	remoteHost := net.JoinHostPort(rmtHost, rmtPort)

	err = targetConn.createClient(remoteHost, rmtUser, rmtPass)
	if err != nil {
		log.Fatal("remote host connect failed with error: ", err)
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
		localCopy(targetConn.Client, jmpHost, flag.Args())
	}

}

func (c *Connect) createClient(host, user, password string) (err error) {

	// Create new ssh.ClientConfig{}

	timeout := 20
	if c.ConnectTimeout > 0 {
		timeout = c.ConnectTimeout
	}

	config := makeClientConfig(user, password, timeout)

	// check Dialer
	if c.ProxyDialer == nil {
		c.ProxyDialer = proxy.Direct
	}

	// Dial to host:port
	netConn, err := c.ProxyDialer.Dial("tcp", host)
	if err != nil {
		return
	}

	// Create new ssh connect
	sshCon, channel, req, err := ssh.NewClientConn(netConn, host, &config)
	if err != nil {
		return
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
		return
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
