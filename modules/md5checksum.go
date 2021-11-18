package modules

import (
	"crypto/md5"
	"io"
	"log"
	"os"
)

// MD5Sum check file's md5 value
func MD5Sum(filePath string) []byte {
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
	return h.Sum(nil)
}
