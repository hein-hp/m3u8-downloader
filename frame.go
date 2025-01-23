package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func ParseFrame(body, baseUrlPrefix string) (frames []Frame) {
	for _, v := range strings.Split(body, "\n") {
		if strings.HasPrefix(v, "#") || strings.TrimSpace(v) == "" {
			continue
		}
		name, err := getFileNameFromUrl(v)
		if err != nil {
			log.Fatalf("无法解析文件名: %v", err)
		}
		toUrl := v
		if !strings.HasPrefix(v, "http") {
			toUrl = fmt.Sprintf("%s/%s", baseUrlPrefix, v)
		}
		frames = append(frames, Frame{
			Name: name,
			Url:  toUrl,
		})
	}
	return frames
}

func MergeFrame(source M3U8, targetDir, fileName string) error {
	tempM3u8 := path.Join(targetDir, fileName+".m3u8")
	targetMp4 := path.Join(targetDir, fileName+".mp4")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	tempDir := filepath.Join(homeDir, ".m3u8_temp", fileName)

	outFile, err := os.OpenFile(tempM3u8, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("创建输出文件 %s 失败: %v", tempM3u8, err)
	}
	defer func() { _ = outFile.Close() }()

	for _, t := range source.frames {
		framePath := filepath.Join(tempDir, t.Name)
		frameData, err := os.ReadFile(framePath)
		if err != nil {
			return fmt.Errorf("读取帧文件失败 %s: %v", framePath, err)
		}
		_, err = outFile.Write(frameData)
		if err != nil {
			return fmt.Errorf("写入帧文件失败 %s: %v", framePath, err)
		}
		err = os.Remove(framePath)
		if err != nil {
			log.Printf("删除帧文件失败 %s: %v", framePath, err)
		}
	}

	// 使用ffmpeg转换为mp4
	cmd := exec.Command("ffmpeg", "-i", tempM3u8, "-c", "copy", targetMp4)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("转换MP4失败: %v", err)
	}

	cleanTempFiles(tempDir, tempM3u8)
	return err
}

func cleanTempFiles(tempDir, tempM3u8 string) {
	retry := func(action func() error) error {
		maxRetries := 10
		for i := 0; i < maxRetries; i++ {
			if err := action(); err == nil {
				return nil
			}
			time.Sleep(time.Second * time.Duration(i+1)) // 指数退避
		}
		return fmt.Errorf("删除失败，已达到最大重试次数")
	}
	// 清理临时目录和m3u8文件
	_ = retry(func() error { return os.RemoveAll(tempDir) })
	_ = retry(func() error { return os.Remove(tempM3u8) })
}

func getFileNameFromUrl(input string) (string, error) {
	parsed, err := url.Parse(input)
	if err != nil {
		return "", fmt.Errorf("invalid URL or path: %v", err)
	}

	var pathPart string
	if parsed.Scheme == "" && !strings.HasPrefix(input, "/") {
		pathPart = input
	} else {
		pathPart = parsed.Path
	}
	fileName := path.Base(pathPart)
	if fileName == "" || fileName == "/" {
		return "", fmt.Errorf("no valid file name found in the input")
	}

	if !strings.HasSuffix(fileName, ".ts") {
		fileName = strings.TrimSuffix(fileName, path.Ext(fileName)) + ".ts"
	}

	return fileName, nil
}
