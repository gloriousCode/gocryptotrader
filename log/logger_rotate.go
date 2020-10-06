package log

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/file"
)

func SetFilePath(fp string) {
	rwm.Lock()
	filePath = fp
	rwm.Unlock()
}

func FilePath() string {
	var resp string
	rwm.RLock()
	defer rwm.RUnlock()
	resp = filePath
	return resp
}

// Write implementation to satisfy io.Writer handles length check and rotation
func (r *Rotate) Write(output []byte) (n int, err error) {
	rwm.Lock()
	defer rwm.Unlock()

	outputLen := int64(len(output))

	if outputLen > r.calculateMaxSize() {
		return 0, fmt.Errorf(
			"write length %v exceeds max file size %v", outputLen, r.calculateMaxSize(),
		)
	}

	if r.output == nil {
		err = r.openOrCreateFile(outputLen)
		if err != nil {
			return 0, err
		}
	}

	if *r.rotationEnabled {
		if r.size+outputLen > r.calculateMaxSize() {
			err = r.rotateFile()
			if err != nil {
				return 0, err
			}
		}
	}

	n, err = r.output.Write(output)
	r.size += int64(n)

	return n, err
}

func (r *Rotate) openOrCreateFile(n int64) error {
	rwm.Lock()
	defer rwm.Unlock()
	logFile := filepath.Join(filePath, r.fileName)

	info, err := os.Stat(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return r.openNew()
		}
		return fmt.Errorf("error opening log file info: %s", err)
	}

	if *r.rotationEnabled {
		if info.Size()+n >= r.calculateMaxSize() {
			return r.rotateFile()
		}
	}
	var f *os.File
	f, err = os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return r.openNew()
	}

	r.output = f
	r.size = info.Size()
	return nil
}

func (r *Rotate) openNew() error {
	rwm.Lock()
	defer rwm.Unlock()
	name := filepath.Join(filePath, r.fileName)
	_, err := os.Stat(name)

	if err == nil {
		timestamp := time.Now().Format("2006-01-02T15-04-05")
		newName := filepath.Join(filePath, timestamp+"-"+r.fileName)

		err = file.Move(name, newName)
		if err != nil {
			return fmt.Errorf("can't rename log file: %s", err)
		}
	}

	var f *os.File
	f, err = os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("can't open new logfile: %s", err)
	}

	r.output = f
	r.size = 0

	return nil
}

func (r *Rotate) close() (err error) {
	rwm.Lock()
	defer rwm.Unlock()
	if r.output == nil {
		return nil
	}
	err = r.output.Close()
	r.output = nil
	return err
}

// Close handler for open file
func (r *Rotate) Close() error {
	return r.close()
}

func (r *Rotate) rotateFile() (err error) {
	err = r.close()
	if err != nil {
		return
	}

	err = r.openNew()
	if err != nil {
		return
	}
	return nil
}

func (r *Rotate) calculateMaxSize() int64 {
	if r.maxSize == 0 {
		return int64(defaultMaxSize * megabyte)
	}
	return r.maxSize * int64(megabyte)
}
