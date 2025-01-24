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
)

// ParseFrame 解析帧方法
// 需要从m3u8文件中，找出每一帧实际的ts文件，这里复杂的地方在于很多网站都对ts文件进行了伪装
// 比如伪装为png,jepg,jpg文件，再比如拿到的是一个请求路径，需要再次请求才能获取资源
func ParseFrame(body, host, baseUrlPrefix string) (frames []Frame) {
	lines := strings.Split(body, "\n")
	for _, v := range lines {
		if strings.HasPrefix(v, "#") || strings.TrimSpace(v) == "" {
			continue
		}
		name, err := getFileNameFromUrl(v)
		if err != nil {
			log.Fatalf("无法解析文件名: %v", err)
		}
		toUrl := ""
		switch isPathOrResource(v) {
		case "resource":
			toUrl = v
			if !strings.HasPrefix(v, "http") {
				toUrl = fmt.Sprintf("%s/%s", baseUrlPrefix, v)
			}
		case "path":
			toUrl = fmt.Sprintf("https://%s/%s", host, v)
		default:
			log.Fatalf("无法识别的文件类型: %s", v)
		}
		frames = append(frames, Frame{
			Name: name,
			Url:  toUrl,
		})
	}
	return frames
}

func MergeFrame(source M3U8, targetDir, fileName string) error {
	// 临时文件：targetDir/fileName.m3u8
	temp := path.Join(targetDir, fileName+".m3u8")
	// 目标文件：targetDir/fileName.mp4
	target := path.Join(targetDir, fileName+".mp4")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// 临时文件夹：用户目录/.m3u8_temp/fileName
	tempDir := filepath.Join(homeDir, ".m3u8_temp", fileName)

	outFile, err := os.OpenFile(temp, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("创建输出文件 %s 失败: %v", temp, err)
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
	}

	// 使用ffmpeg转换为mp4
	cmd := exec.Command("ffmpeg", "-i", temp, "-c", "copy", "-movflags", "+faststart", target)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("转换MP4失败: %v", err)
	}

	// 清除临时文件
	cleanTempFiles(tempDir, temp)

	return nil
}

func cleanTempFiles(tempDir, temp string) {
	// 清理临时目录和m3u8文件
	err := os.RemoveAll(tempDir)
	if err != nil {
		log.Fatalf("删除临时目录失败: %v", err)
	}
	err = os.Remove(temp)
	if err != nil {
		log.Fatalf("删除临时m3u8文件失败: %v", err)
	}
}

// getFileNameFromUrl 从URL中获取文件名
// 获取规则是拿到url的最后一段，如果非.ts结尾，则替换为.ts
// "http://example.com/path/to/file.js",
// "http://example.com/path/to/file",
// "relative/path/to/file.mp4",
// "relative/path/to/file",
// "/absolute/path/to/file.png",
// "/absolute/path/to/file", 最后结果都是file.ts
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

// isPathOrResource 是请求路径还是资源路径
// 请求路径：https://4k.tv/cascascsafaffq
// 资源路径：https://4k.tv/cascascsafaffq/1.ts
func isPathOrResource(input string) string {
	parse, err := url.Parse(input)
	if err != nil {
		log.Fatalf("解析URL失败: %v", err)
	}
	p := parse.Path
	ext := filepath.Ext(p)
	if ext != "" && ext != "." {
		return "resource"
	}
	if strings.HasSuffix(p, "/") || ext == "" {
		return "path"
	}
	return "unknown"
}
