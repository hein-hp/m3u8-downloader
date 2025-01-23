package main

import (
	"fmt"
	"github.com/schollz/progressbar/v3"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func Download(source M3U8, maxGoroutines int64, output string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("获取用户主目录失败: %v", err)
	}

	tempDir := filepath.Join(homeDir, ".m3u8_temp", output)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Fatalf("创建临时目录失败: %v", err)
	}

	retry := 100
	var wg sync.WaitGroup
	limiter := make(chan struct{}, maxGoroutines)

	// 添加进度统计
	totalFiles := len(source.frames)
	bar := progressbar.NewOptions(totalFiles,
		progressbar.OptionSetDescription("下载中"),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	for _, frame := range source.frames {
		wg.Add(1)
		limiter <- struct{}{}
		go func(frame Frame, tempDir string, retry int) {
			defer func() {
				wg.Done()
				<-limiter
			}()
			doDown(frame, tempDir, source.encrypt, retry)
			_ = bar.Add(1)
		}(frame, tempDir, retry)
	}
	wg.Wait()

	// 统计下载帧数量
	files, err := countFiles(tempDir)
	if err != nil {
		log.Fatalf("统计文件失败: %v", err)
	}
	percentage := (float64(files) / float64(totalFiles)) * 100
	log.Printf("共 %d 帧文件, 已下载 %d 帧文件, 已下载 %.2f%", totalFiles, files, percentage)
	if totalFiles != files {
		log.Printf("请降低并发量，然后重试")
	}
}

func doDown(frame Frame, dir string, encrypt Encrypt, retry int) {
	if retry < 0 {
		log.Printf("当前文件 %s 下载失败", frame.Url)
		return
	}
	current := filepath.Join(dir, frame.Name)
	if isExist, _ := pathExists(current); isExist {
		log.Printf("文件 %s 已存在", frame.Name)
		return
	}
	resp, err := HttpGet(&HttpRequestConfig{
		URL: frame.Url,
	})
	if err != nil {
		if retry > 0 {
			time.Sleep(500 * time.Millisecond) // 休眠，防止频繁请求
			doDown(frame, dir, encrypt, retry-1)
			return
		} else {
			log.Printf("文件 %s 下载失败: %v", frame.Name, err)
			return
		}
	}

	if encrypt.method != "" && encrypt.method != "NONE" {
		data, err := AESDecrypt([]byte(resp), []byte(encrypt.key), encrypt.iv)
		if err != nil {
			log.Printf("解密失败: %v", err)
			doDown(frame, dir, encrypt, retry-1)
			return
		}
		syncByte := uint8(71) // 0x47
		bLen := len(data)
		for j := 0; j < bLen; j++ {
			if data[j] == syncByte {
				data = data[j:]
				break
			}
		}
		err = os.WriteFile(current, data, 0666)
		if err != nil {
			log.Printf("写入文件失败: %v", err)
			time.Sleep(500 * time.Millisecond) // 休眠，防止频繁请求
			doDown(frame, dir, encrypt, retry-1)
			return
		}
	} else {
		err = os.WriteFile(current, []byte(resp), 0666)
		if err != nil {
			log.Printf("写入文件失败: %v", err)
			time.Sleep(500 * time.Millisecond) // 休眠，防止频繁请求
			doDown(frame, dir, encrypt, retry-1)
			return
		}
	}
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func countFiles(dir string) (int, error) {
	var count int
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			count++
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("error walking the path %q: %v", dir, err)
	}
	return count, nil
}
