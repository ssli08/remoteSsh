package modules

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/pkg/sftp"
	"github.com/schollz/progressbar/v3"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
	"golang.org/x/crypto/ssh"
)

type FileTransfer struct {
	// pool      chan *Resource
	wg        *sync.WaitGroup
	DestPath  string
	Resources []Resource
	Conn      *ssh.Client
}

type Resource struct {
	Filename string
	LFileMD5 string
}

func newFileTransfer(destpath string, conn *ssh.Client) *FileTransfer {
	return &FileTransfer{
		wg:       &sync.WaitGroup{},
		DestPath: destpath,
		Conn:     conn,
	}
}
func (f *FileTransfer) appendResources(filename string) {
	lmd5 := MD5Sum(filename)
	f.Resources = append(f.Resources,
		Resource{
			Filename: filename,
			LFileMD5: lmd5,
		})
}

func (f *FileTransfer) transfer(rs Resource, p *mpb.Progress) Resource {
	defer f.wg.Done()
	// f.pool <- &rs
	sc, err := sftp.NewClient(f.Conn)
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()
	if f.DestPath == "" {
		d, _ := sc.Getwd()
		f.DestPath = d
	}
	targetFileName := path.Join(f.DestPath, path.Base(rs.Filename))
	if _, err := sc.Lstat(targetFileName); err != nil {
		// log.Println(err)
		fmt.Printf("%s, start copying file %s to %s\n", err, rs.Filename, f.Conn.RemoteAddr())
	} else {
		rmd5, err := RMD5Sum(f.Conn, targetFileName)
		if err != nil {
			// log.Fatal(err)
			fmt.Println(err)
		}
		log.Printf("file %s' md5 local: (%s) remote: (%s)\n", rs.Filename, rs.LFileMD5, rmd5)
		if rs.LFileMD5 == rmd5 {
			// fmt.Printf("same md5 value for [%s] between  Local and Remote\n", rs.Filename)
			fmt.Printf("Local file (%s) has the same MD5 value as the Remote, nothing to do\n", rs.Filename)
			return Resource{}
		} else {
			fmt.Printf("get different md5 value for [%s] on remote server, start copying..\n", targetFileName)
			sc.Rename(targetFileName, strings.Join([]string{targetFileName, time.Now().Format("20060102-150405")}, "-"))
		}
	}
	rf, err := sc.Create(targetFileName)
	if err != nil {
		log.Fatalf("create file %s failed on server %s", targetFileName, f.Conn.RemoteAddr())
	}
	defer rf.Close()

	if err = rf.Chmod(0755); err != nil {
		log.Fatal("change permission failed with error ", err)
	}

	buf, err := os.Open(rs.Filename)
	if err != nil {
		log.Fatal(err)
	}
	defer buf.Close()

	st := time.Now()
	t, _ := buf.Stat()
	// bar := progressbar.DefaultBytes(t.Size(), fmt.Sprintf("transferring file %s ", t.Name()))
	// bar := progressBarDef(t.Size(), fmt.Sprintf("transferring file %s ", t.Name()))
	// io.Copy(io.MultiWriter(rf, bar), buf)
	bar := p.AddBar(
		int64(t.Size()),
		// 进度条前的修饰
		mpb.PrependDecorators(
			decor.Name(fmt.Sprintf("[ transfer %s ", t.Name()), decor.WC{W: len(fmt.Sprintf("transfer %s ", t.Name())) + 1, C: decor.DidentRight}),
			decor.CountersKibiByte("% .2f / % .2f"), // 已下载数量
			decor.Percentage(decor.WCSyncSpace),     // 进度百分比
		),
		// 进度条后的修饰
		mpb.AppendDecorators(
			// decor.EwmaETA(decor.ET_STYLE_GO, 90),
			decor.AverageETA(decor.ET_STYLE_GO),
			decor.Name(" ] "),
			// decor.EwmaSpeed(decor.UnitKiB, "% .2f", 60),
			decor.AverageSpeed(decor.UnitKiB, "% .2f"),
			// decor.OnComplete(
			// 	decor.AverageETA(decor.ET_STYLE_GO, decor.WC{W: 4}), "done",
			// ),
		),
	)
	reader := bar.ProxyReader(buf)
	defer reader.Close()
	if _, err := io.Copy(rf, reader); err != nil {
		log.Fatal(err)
	}

	duration := time.Since(st)
	log.Printf("Copy [File: %s, MD5: %s ] to %s [remote path: %s] successfully in %s.\n",
		path.Base(rs.Filename), rs.LFileMD5, f.Conn.RemoteAddr(), targetFileName, duration)
	return rs
	// <-f.pool
}
func (f *FileTransfer) start() {
	// f.pool = make(chan *Resource, runtime.NumCPU())
	result := []Resource{}

	p := mpb.New(mpb.WithWaitGroup(f.wg), mpb.WithRefreshRate(10*time.Millisecond))
	for _, resource := range f.Resources {
		// fmt.Printf("File: %s, MD5: %s\n\n", resource.Filename, resource.LFileMD5)
		f.wg.Add(1)
		go func(rs Resource) {
			res := f.transfer(rs, p)
			result = append(result, res)
		}(resource)
	}
	p.Wait()
	f.wg.Wait()

	// print filename and md5 value
	for _, i := range result {
		fmt.Printf("File: %s, MD5: %s\n", i.Filename, i.LFileMD5)
	}
}

func localCopy(conn *ssh.Client, destPath string, lfilePath []string) {
	f := newFileTransfer(destPath, conn)
	for _, fn := range lfilePath {
		f.appendResources(fn)
	}
	f.start()

}

// 3rd part progresssbar for `progresssbar` lib
func progressBarDef(maxSize int64, desc string) *progressbar.ProgressBar {

	bar := progressbar.NewOptions64(
		maxSize,
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetDescription(desc),
		progressbar.OptionSetWidth(50),
		// progressbar.OptionEnableColorCodes(true),

	)
	return bar
}

// 3rd part single progresssbar for `pb` lib
func exPB(filename string, buf io.Reader, rf io.Writer) {
	t, _ := os.Stat(filename)
	bar := pb.New(int(t.Size())).SetUnits(pb.U_BYTES).SetRefreshRate(10 * time.Millisecond)
	defer bar.Finish()
	bar.Prefix(t.Name())
	bar.ShowSpeed = true
	bar.ShowCounters = true
	bar.ShowTimeLeft = true
	bar.SetWidth(60)
	bar.Start()

	reader := bar.NewProxyReader(buf)
	if _, err := io.Copy(rf, reader); err != nil {
		log.Fatal(err)
	}
	bar.Finish()
}

// 3rd part multiple progresssbar for `pb` lib
func exMultiplePB() {
	// create bars
	first := pb.New(200).Prefix("First ")
	second := pb.New(200).Prefix("Second ")
	third := pb.New(200).Prefix("Third ")
	// start pool
	pool, err := pb.StartPool(first, second, third)
	if err != nil {
		panic(err)
	}
	// update bars
	wg := new(sync.WaitGroup)
	for _, bar := range []*pb.ProgressBar{first, second, third} {
		wg.Add(1)
		go func(cb *pb.ProgressBar) {
			for n := 0; n < 200; n++ {
				cb.Increment()
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
			}
			cb.Finish()
			wg.Done()
		}(bar)
	}
	wg.Wait()
	// close pool
	pool.Stop()
}
