package modules

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gosuri/uiprogress"
	"github.com/pkg/sftp"
	"github.com/schollz/progressbar/v3"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
	"golang.org/x/crypto/ssh"
)

type FileTransfer struct {
	pool      chan *Resource
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
func (f *FileTransfer) transfer(rs Resource, p *mpb.Progress) {
	defer f.wg.Done()
	f.pool <- &rs
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
			fmt.Printf("same md5 value for [%s] between  Local and Remote\n", rs.Filename)
			return
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
			decor.Name(fmt.Sprintf("transfer %s.. ", t.Name())),
			decor.CountersKibiByte("% .2f / % .2f"), // 已下载数量
			decor.Percentage(decor.WCSyncSpace),     // 进度百分比
		),
		// 进度条后的修饰
		mpb.AppendDecorators(
			decor.EwmaETA(decor.ET_STYLE_GO, 90),
			decor.Name(" ] "),
			decor.EwmaSpeed(decor.UnitKiB, "% .2f", 60),
		),
	)
	reader := bar.ProxyReader(buf)
	defer reader.Close()
	if _, err := io.Copy(rf, reader); err != nil {
		log.Fatal(err)
	}
	duration := time.Since(st)

	fmt.Printf("Copy [File: %s, MD5: %s ] to %s [remote path: %s] successfully in %s.\n",
		path.Base(rs.Filename), rs.LFileMD5, f.Conn.RemoteAddr(), targetFileName, duration)

	<-f.pool
}
func (f *FileTransfer) start() {
	f.pool = make(chan *Resource, runtime.NumCPU())
	p := mpb.New(mpb.WithWaitGroup(f.wg))
	for _, resource := range f.Resources {
		f.wg.Add(1)
		go f.transfer(resource, p)
	}
	p.Wait()
	f.wg.Wait()
}
func localCopy(conn *ssh.Client, destPath string, lfilePath []string) {
	f := newFileTransfer(destPath, conn)
	for _, fn := range lfilePath {
		f.appendResources(fn)
	}
	f.start()

}

/* func localCopy(conn *ssh.Client, destPath string, lfilePath []string) {

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

	p := mpb.New(
		mpb.WithWaitGroup(&wg),
		mpb.WithWidth(60),
		// mpb.WithRefreshRate(180*time.Millisecond),
	)
	defer p.Wait()
	defer wg.Wait()

	if len(lfilePath) > 0 {
		for _, file := range lfilePath {
			wg.Add(1)

			// fmt.Printf("start copying file %s to %s\n", fileName, conn.RemoteAddr())

			go writeRemoteFile(file, destPath, s, sftpClient, conn.RemoteAddr(), p, &wg)

		}
	} else {
		log.Fatal("no file provided, exit.")
	}

}

func writeRemoteFile(lfilePath, destPath string, session *ssh.Session, sftpClient *sftp.Client, raddr net.Addr, progress *mpb.Progress, wg *sync.WaitGroup) {
	defer wg.Done()
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
		rm, err := RMD5Sum(session, fileName)
		if err != nil {
			// log.Fatal(err)
			fmt.Println(err)
		}
		log.Printf("file %s' md5 local: (%s) remote: (%s)\n", lfilePath, lm, rm)
		if lm == rm {
			fmt.Printf("same md5 value for [%s] between  Local and Remote\n", lfilePath)
			return
		} else {
			fmt.Printf("different md5 value for [%s] on remote server, start copying..\n", fileName)
			// sftpClient.Rename(fileName, strings.Join([]string{fileName, time.Now().Format("20060102-150405")}, "-"))
		}
	}

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
	// bar := progressbar.DefaultBytes(a.Size(), fmt.Sprintf("transferring file %s ", a.Name()))
	// bar := progressBarDef(a.Size(), fmt.Sprintf("transferring file %s ", a.Name()))
	// io.Copy(io.MultiWriter(f, bar), buf)
	bar := progress.AddBar(
		int64(a.Size()),
		// 进度条前的修饰
		mpb.PrependDecorators(
			// decor.Name(fmt.Sprintf("transfer %s", a.Name())),
			decor.CountersKibiByte("% .2f / % .2f"), // 已下载数量
			decor.Percentage(decor.WCSyncSpace),     // 进度百分比
		),
		// 进度条后的修饰
		mpb.AppendDecorators(
			decor.EwmaETA(decor.ET_STYLE_GO, 90),
			decor.Name(" ] "),
			decor.EwmaSpeed(decor.UnitKiB, "% .2f", 60),
		),
	)

	reader := bar.ProxyReader(buf)
	defer reader.Close()
	if _, err := io.Copy(f, reader); err != nil {
		log.Fatal(err)
	}

	duration := time.Since(st)
	fmt.Printf("Copy [File: %s, MD5: %s ] to %s [remote path: %s] successfully in %s.\n", path.Base(lfilePath), lm, raddr.String(), fileName, duration)
} */

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
func UIProgressBar() {
	waitTime := time.Millisecond * 100
	uiprogress.Start()
	// start the progress bars in go routines
	var wg sync.WaitGroup

	bar1 := uiprogress.AddBar(20).AppendCompleted().PrependElapsed()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for bar1.Incr() {
			time.Sleep(waitTime)
		}
	}()

	bar2 := uiprogress.AddBar(40).AppendCompleted().PrependElapsed()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for bar2.Incr() {
			time.Sleep(waitTime)
		}
	}()

	time.Sleep(time.Second)
	bar3 := uiprogress.AddBar(20).PrependElapsed().AppendCompleted()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= bar3.Total; i++ {
			bar3.Set(i)
			time.Sleep(waitTime)
		}
	}()
	// wait for all the go routines to finish
	wg.Wait()
}
func multiprogressBar(totalSize int64, filename *os.File, progress *mpb.Progress) {
	bar := progress.AddBar(
		totalSize,

		// decorator before progress bar
		mpb.PrependDecorators(
			decor.CountersKibiByte("% .2f / % .2f"), // downloaded amount
			decor.Percentage(decor.WCSyncSpace),     // progress percentage
		),
		// decorator after progress bar
		mpb.AppendDecorators(
			decor.EwmaETA(decor.ET_STYLE_GO, 90),
			decor.Name(" ] "),
			decor.EwmaSpeed(decor.UnitKiB, "% .2f", 60),
		),
	)

	reader := bar.ProxyReader(filename)
	defer reader.Close()

}
func MultiProgressBarPresentation(total int64, fileNum int) {
	var wg sync.WaitGroup
	// passed &wg will be accounted at p.Wait() call
	p := mpb.New(mpb.WithWaitGroup(&wg))
	wg.Add(fileNum)

	for i := 0; i < fileNum; i++ {
		name := fmt.Sprintf("transferring %d:", i)
		bar := p.AddBar(total,
			mpb.PrependDecorators(
				// simple name decorator
				decor.Name(name),
				// decor.DSyncWidth bit enables column width synchronization
				decor.Percentage(decor.WCSyncSpace),
			),
			mpb.AppendDecorators(
				// replace ETA decorator with "done" message, OnComplete event
				decor.OnComplete(
					// ETA decorator with ewma age of 60
					decor.EwmaETA(decor.ET_STYLE_GO, 60, decor.WCSyncWidth), "done",
				),
			),
		)
		// simulating some work
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewSource(time.Now().UnixNano()))
			max := 100 * time.Millisecond
			for i := 0; i < int(total); i++ {
				// start variable is solely for EWMA calculation
				// EWMA's unit of measure is an iteration's duration
				start := time.Now()
				time.Sleep(time.Duration(rng.Intn(10)+1) * max / 10)
				bar.Increment()
				// we need to call DecoratorEwmaUpdate to fulfill ewma decorator's contract
				bar.DecoratorEwmaUpdate(time.Since(start))
			}
		}()
	}
	// Waiting for passed &wg and for all bars to complete and flush
	p.Wait()

}
