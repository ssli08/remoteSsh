package modules

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/crypto/ssh"
)

func localCopy(conn *ssh.Client, destPath string, lfilePath []string) {

	var wg sync.WaitGroup

	sftpClient, err := sftp.NewClient(conn)
	if err != nil {
		log.Fatal(err)
	}
	defer sftpClient.Close()

	s, err := conn.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	defer wg.Wait()

	if len(lfilePath) > 0 {
		for _, file := range lfilePath {
			wg.Add(1)
			// log.Printf("start copying file %s to %s", file, conn.RemoteAddr())
			go func(fileName string, wg *sync.WaitGroup) {
				// fmt.Printf("start copying file %s to %s\n", fileName, conn.RemoteAddr())
				defer wg.Done()
				writeRemoteFile(fileName, destPath, s, sftpClient, conn.RemoteAddr())
			}(file, &wg)

		}
	} else {
		log.Fatal("no file provided, exit.")
	}

}

func writeRemoteFile(lfilePath, destPath string, session *ssh.Session, sftpClient *sftp.Client, raddr net.Addr) {

	lm := MD5Sum(lfilePath)

	if destPath == "" {
		d, _ := sftpClient.Getwd()
		destPath = d
	}
	fileName := path.Join(destPath, path.Base(lfilePath))

	if _, err := sftpClient.Lstat(fileName); err != nil {
		// log.Println(err)
		fmt.Printf("%s, start copying file %s to %s\n", err, lfilePath, raddr)
	} else {
		rm := RMD5Sum(session, fileName)
		log.Printf("file %s' md5 local: (%s) remote: (%s)\n", lfilePath, lm, rm)
		if lm == rm {
			fmt.Printf("same md5 value for [%s] between  Local and Remote\n", lfilePath)
			return
		} else {
			fmt.Printf("different md5 value for [%s] on remote server, start copying..", fileName)
			sftpClient.Rename(fileName, strings.Join([]string{fileName, time.Now().Format("20060102-150405")}, "-"))
		}
	}

	// fileName := path.Base(filePath)

	f, err := sftpClient.Create(fileName)
	if err != nil {
		log.Fatalf("create file %s failed on server %s", fileName, raddr)
	}
	defer f.Close()

	if err = f.Chmod(0755); err != nil {
		log.Fatal("change permission failed with error ", err)
	}

	buf, err := os.Open(lfilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer buf.Close()

	st := time.Now()
	// if _, err := f.ReadFrom(buf); err != nil {
	// 	log.Fatal("read error ", err)
	// }

	// progress bar setting
	a, _ := buf.Stat()
	// bar := progressbar.DefaultBytes(a.Size(), "transferring")
	bar := progressBarDef(a.Size(), fmt.Sprintf("transferring file %s ", a.Name()))
	io.Copy(io.MultiWriter(f, bar), buf)

	duration := time.Since(st)
	fmt.Printf("Copy [File: %s, MD5: %s ] to %s [remote path: %s] successfully in %s.\n", path.Base(lfilePath), lm, raddr.String(), destPath, duration)
}

func progressBarDef(maxSize int64, desc string) *progressbar.ProgressBar {
	bar := progressbar.NewOptions64(
		maxSize,
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetDescription(desc),
		progressbar.OptionSetWidth(50),
		// progressbar.OptionClearOnFinish(),
		progressbar.OptionOnCompletion(func() { os.Stdout.Write([]byte("\n")) }),
	)
	return bar
}
