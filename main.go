package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"path"
	"strconv"
)

func main() {

	homeDir, _ := os.UserHomeDir()
	urlFlag := flag.String("l", "", "下载链接")
	dirFlag := flag.String("d", homeDir, "目标目录")
	outputFlag := flag.String("o", "output", "输出名称")
	parallelFlag := flag.String("p", "50", "并发数")

	// 解析命令行参数
	flag.Parse()

	if *urlFlag == "" {
		log.Fatal("必须提供 -l 参数")
	}

	// 下载 M3U8 文件
	body, err := HttpGet(&HttpRequestConfig{
		URL: *urlFlag,
	})
	if err != nil {
		log.Fatalf("获取M3U8文件失败: %v", err)
	}

	// 解析 M3U8 文件
	source := Parse(body, *urlFlag)

	// 下载所有帧
	parallel, err := strconv.ParseInt(*parallelFlag, 10, 64)
	if err != nil {
		log.Fatalf("解析并发数失败: %v", err)
	}
	Download(source, parallel, *outputFlag)

	// 合并帧为 MP4 文件
	err = MergeFrame(source, *dirFlag, *outputFlag)
	if err != nil {
		log.Fatalf("合并文件失败: %v", err)
	}

	log.Printf("文件已合并到: %s", path.Join(*dirFlag, *outputFlag+".mp4"))
}

// Parse 解析出M3U8
func Parse(body, baseUrl string) M3U8 {
	base, err := url.Parse(baseUrl)
	if err != nil {
		log.Fatalf("解析URL失败: %v", err)
	}
	base.Path = path.Dir(base.Path)
	baseUrlPrefix := base.String()

	frame := ParseFrame(body, baseUrlPrefix)
	encrypt := ParseEncrypt(body, baseUrlPrefix)
	return M3U8{
		baseUrl:       baseUrl,
		baseUrlPrefix: baseUrlPrefix,
		frames:        frame,
		encrypt:       encrypt,
	}
}
