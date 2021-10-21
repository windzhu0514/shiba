package log

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	backupTimeFormat = "20060102"
	compressSuffix   = ".gz"
	defaultMaxSize   = 100
)

var _ io.WriteCloser = (*dailyRotator)(nil)

type dailyRotator struct {
	Filename  string `json:"filename" yaml:"filename"`
	MaxAge    int    `json:"maxage" yaml:"maxage"`
	LocalTime bool   `json:"localtime" yaml:"localtime"`
	Compress  bool   `json:"compress" yaml:"compress"`

	mu          sync.Mutex
	millCh      chan bool
	startMill   sync.Once
	rotatedDate string
	file        *os.File
}

func (r *dailyRotator) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file == nil {
		if err = r.openExistingOrNew(); err != nil {
			return 0, err
		}
	}

	if len(r.rotatedDate) > 0 && r.rotatedDate != time.Now().Format(backupTimeFormat) {
		if err := r.rotate(); err != nil {
			return 0, err
		}

		r.rotatedDate = time.Now().Format(backupTimeFormat)
	}

	n, err = r.file.Write(p)

	return n, nil
}

func (r *dailyRotator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.close()
}

func (r *dailyRotator) close() error {
	if r.file == nil {
		return nil
	}

	err := r.file.Close()
	r.file = nil
	return err
}

func (r *dailyRotator) rotate() error {
	if err := r.close(); err != nil {
		return err
	}

	if err := r.openNew(); err != nil {
		return err
	}

	r.mill()

	return nil
}

func (r *dailyRotator) openExistingOrNew() error {
	r.mill()

	r.rotatedDate = time.Now().Format(backupTimeFormat)

	fileName := r.filename()
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return r.openNew()
	}

	if err != nil {
		return fmt.Errorf("error getting log file info: %s", err)
	}

	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return r.openNew()
	}
	r.file = file
	return nil
}

func (r *dailyRotator) millRunOnce() error {
	if r.MaxAge == 0 && !r.Compress {
		return nil
	}

	files, err := r.oldLogFiles()
	if err != nil {
		return err
	}

	var compress, remove []logInfo
	if r.MaxAge > 0 {
		diff := time.Duration(int64(24*time.Hour) * int64(r.MaxAge))
		cutoff := time.Now().Add(-1 * diff)

		var remaining []logInfo
		for _, f := range files {
			if f.timestamp.Before(cutoff) {
				remove = append(remove, f)
			} else {
				remaining = append(remaining, f)
			}
		}
		files = remaining
	}

	if r.Compress {
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), compressSuffix) {
				compress = append(compress, f)
			}
		}
	}

	for _, f := range remove {
		errRemove := os.Remove(filepath.Join(r.dir(), f.Name()))
		if err == nil && errRemove != nil {
			err = errRemove
		}
	}

	for _, f := range compress {
		fn := filepath.Join(r.dir(), f.Name())
		errCompress := compressLogFile(fn, fn+compressSuffix)
		if err == nil && errCompress != nil {
			err = errCompress
		}
	}

	return err
}

func compressLogFile(src, dst string) (err error) {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	defer f.Close()

	fi, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat log file: %v", err)
	}

	if err := os.Chown(src, os.Getuid(), os.Getgid()); err != nil {
		return fmt.Errorf("failed to chown compressed log file: %v", err)
	}

	gzf, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fi.Mode())
	if err != nil {
		return fmt.Errorf("failed to open compressed log file: %v", err)
	}
	defer gzf.Close()

	gz := gzip.NewWriter(gzf)

	defer func() {
		if err != nil {
			os.Remove(dst)
			err = fmt.Errorf("failed to compress log file: %v", err)
		}
	}()

	if _, err = io.Copy(gz, f); err != nil {
		return err
	}
	if err = gz.Close(); err != nil {
		return err
	}

	if err = gzf.Close(); err != nil {
		return err
	}

	if err = f.Close(); err != nil {
		return err
	}

	if err = os.Remove(src); err != nil {
		return err
	}

	return nil
}

type logInfo struct {
	timestamp time.Time
	os.FileInfo
}

func (r *dailyRotator) oldLogFiles() ([]logInfo, error) {
	files, err := ioutil.ReadDir(r.dir())
	if err != nil {
		return nil, fmt.Errorf("can't read log file directory: %s", err)
	}

	var logFiles []logInfo

	prefix, ext := r.prefixAndExt()

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		if t, err := r.timeFromName(f.Name(), prefix, ext); err == nil {
			logFiles = append(logFiles, logInfo{t, f})
			continue
		}
		if t, err := r.timeFromName(f.Name(), prefix, ext+compressSuffix); err == nil {
			logFiles = append(logFiles, logInfo{t, f})
			continue
		}
	}

	sort.Sort(byFormatTime(logFiles))

	return logFiles, nil
}

func (r *dailyRotator) millRun() {
	for _ = range r.millCh {
		// what am I going to do, log this?
		_ = r.millRunOnce()
	}
}

func (r *dailyRotator) mill() {
	r.startMill.Do(func() {
		r.millCh = make(chan bool, 1)
		go r.millRun()
	})
	select {
	case r.millCh <- true:
	default:
	}
}

func (r *dailyRotator) filename() string {
	if r.Filename != "" {
		return r.Filename
	}

	dateStr := time.Now().Format(backupTimeFormat)
	name := filepath.Base(os.Args[0]) + "-" + dateStr + ".log"
	return filepath.Join(os.TempDir(), name)
}

func (r *dailyRotator) openNew() error {
	err := os.MkdirAll(r.dir(), 0744)
	if err != nil {
		return fmt.Errorf("can't make directories for new logfile: %s", err)
	}

	name := r.filename()
	mode := os.FileMode(0644)
	info, err := os.Stat(name)
	if err == nil {
		mode = info.Mode()
		newName := backupName(name, r.LocalTime)
		if err := os.Rename(name, newName); err != nil {
			return fmt.Errorf("can't rename log file: %s", err)
		}

		if err := os.Chown(name, os.Getuid(), os.Getgid()); err != nil {
			return err
		}
	}

	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("can't open new logfile: %s", err)
	}

	r.file = f
	return nil
}

func (r *dailyRotator) dir() string {
	return filepath.Dir(r.filename())
}

func (r *dailyRotator) prefixAndExt() (prefix, ext string) {
	filename := filepath.Base(r.filename())
	ext = filepath.Ext(filename)
	prefix = filename[:len(filename)-len(ext)] + "-"
	return prefix, ext
}

func backupName(name string, local bool) string {
	dir := filepath.Dir(name)
	filename := filepath.Base(name)
	ext := filepath.Ext(filename)
	prefix := filename[:len(filename)-len(ext)]
	t := time.Now()
	if !local {
		t = t.UTC()
	}
	t = t.Add(-time.Hour * 24)

	timestamp := t.Format(backupTimeFormat)
	return filepath.Join(dir, fmt.Sprintf("%s-%s%s", prefix, timestamp, ext))
}

func (r *dailyRotator) timeFromName(filename, prefix, ext string) (time.Time, error) {
	if !strings.HasPrefix(filename, prefix) {
		return time.Time{}, errors.New("mismatched prefix")
	}
	if !strings.HasSuffix(filename, ext) {
		return time.Time{}, errors.New("mismatched extension")
	}
	ts := filename[len(prefix) : len(filename)-len(ext)]
	return time.Parse(backupTimeFormat, ts)
}

type byFormatTime []logInfo

func (b byFormatTime) Less(i, j int) bool {
	return b[i].timestamp.After(b[j].timestamp)
}

func (b byFormatTime) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byFormatTime) Len() int {
	return len(b)
}
