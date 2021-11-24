package modules

import (
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

/*
func remoteCopyProgressBar(session *ssh.Session) {
	fileTotleSize := func() int64 {
		f, err := os.Open(filename)
		if err != nil {
			log.Fatalf("%s not exist", filename)
		}
		fInfo, _ := f.Stat()
		return fInfo.Size()
	}()
	fmt.Println(fileTotleSize)
	fname := fmt.Sprintf(os.Getenv("HOME") + "/" + filename)
	file, err := os.OpenFile(fname, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0755)

	if err != nil {
		log.Fatalf("read error: %s", err)
	}
	r, err := session.StdoutPipe()

	if err != nil {
		log.Fatal(err)
	}
	wn, err := io.Copy(file, r)
	if err != nil {
		log.Fatalf("trasfer file failed with error %s", err)
	}
	fmt.Printf("transfered file size %d, totalFileSize of %s is %d", wn, filename, fileTotleSize)

	if err = session.Wait(); err != nil {
		log.Fatal(err)
	}

}
*/
func localCopy(conn *ssh.Client, jmpHost string, filePath []string) {

	var wg sync.WaitGroup

	sftpClient, err := sftp.NewClient(conn)
	if err != nil {
		log.Fatal(err)
	}
	defer sftpClient.Close()
	defer wg.Wait()

	if len(filePath) > 0 {
		for _, file := range filePath {
			// log.Printf("start copying file %s to %s", file, conn.RemoteAddr())
			fmt.Printf("start copying file %s to %s\n", file, conn.RemoteAddr())
			wg.Add(1)
			go writeRemoteFile(file, jmpHost, sftpClient, conn.RemoteAddr(), &wg)
		}
	} else {
		log.Fatal("no file provided, exit.")
	}
}

func writeRemoteFile(filePath, jmpHost string, sftpClient *sftp.Client, raddr net.Addr, wg *sync.WaitGroup) {

	defer wg.Done()

	fileName := path.Base(filePath)

	f, err := sftpClient.Create(fileName)
	if err != nil {
		log.Fatalf("create file %s failed on server %s", fileName, raddr)
	}
	defer f.Close()

	if err = f.Chmod(0755); err != nil {
		log.Fatal("change permission failed with error ", err)
	}

	buf, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer buf.Close()

	st := time.Now()
	if _, err := f.ReadFrom(buf); err != nil {
		log.Fatal("read error ", err)
	}
	// check time spending during file transportation
	// duration := time.Now().Sub(st)
	duration := time.Since(st)
	// buf, err := ioutil.ReadFile(filePath)
	// if err != nil {
	// 	log.Fatalf("read file error %s", err)
	// }
	// if _, err := f.Write(buf); err != nil {
	// 	log.Fatal(err)
	// }
	md5 := MD5Sum(filePath)
	fmt.Printf("Copy [File: %s, MD5: %x ] to %s successfully in %s.\n", fileName, md5, jmpHost, duration)
}
