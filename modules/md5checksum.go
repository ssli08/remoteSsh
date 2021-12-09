package modules

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"golang.org/x/crypto/ssh"
)

// MD5Sum check file's md5 value
func MD5Sum(filePath string) string {
	// fmt.Printf("%T", buf.Fd())
	buf, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("failed to open file %s", filePath)
	}
	defer buf.Close()

	h := md5.New()
	if _, err := io.Copy(h, buf); err != nil {
		log.Fatalf("failed to check md5 value for file %s, due %s", buf.Name(), err)
	}

	// return base64.StdEncoding.EncodeToString(h.Sum(nil))
	a := fmt.Sprintf("%x", h.Sum(nil))
	// fmt.Println(a)
	return a
}

func RMD5Sum(conn *ssh.Client, filePath string) (string, error) {
	var ebuf, obuf bytes.Buffer

	session, err := conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("new session failed with error %s", err)
	}

	session.Stderr = &ebuf
	session.Stdout = &obuf
	if err := session.Run(fmt.Sprintf("md5sum %s|cut -d ' ' -f1|xargs echo -n", filePath)); err != nil {
		// fmt.Println("rum cmd: ", err)
		// log.Fatal(err)
		return "", fmt.Errorf("rum cmd: %s", err)
	}
	if ebuf.String() != "" {
		return "", errors.New(ebuf.String())
	}
	return obuf.String(), nil
}
