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

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("获取用户主目录失败: %v", err)
	}
	urlFlag := flag.String("l", "", "下载链接")
	dirFlag := flag.String("d", homeDir, "目标目录")
	outputFlag := flag.String("o", "output", "输出名称")
	parallelFlag := flag.String("p", "50", "并发数")
	refererFlag := flag.String("r", "", "Referer")
	cookieFlag := flag.String("c", "", "Cookie")

	// 解析命令行参数
	flag.Parse()

	if *urlFlag == "" {
		log.Fatal("必须提供 -l 参数")
	}

	var ctx Context

	ctx.URL = *urlFlag
	ctx.dir = *dirFlag
	ctx.output = *outputFlag
	ctx.parallel, err = strconv.ParseInt(*parallelFlag, 10, 64)
	if err != nil {
		log.Fatalf("解析并发数失败: %v", err)
	}
	ctx.referer = *refererFlag
	ctx.cookie = *cookieFlag

	// 下载 M3U8 文件
	body, err := HttpGet(&HttpRequestConfig{
		URL: ctx.URL,
		Headers: map[string]string{
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
			"Referer":    ctx.referer,
			"Cookie":     ctx.cookie,
		},
	})
	if err != nil {
		log.Fatalf("获取M3U8文件失败: %v", err)
	}

	// 解析 M3U8 文件
	source := Parse(body, &ctx)

	// 下载所有帧
	Download(source, &ctx)

	// 合并帧为 MP4 文件
	err = MergeFrame(source, &ctx)
	if err != nil {
		log.Fatalf("合并文件失败: %v", err)
	}

	log.Printf("文件已合并到: %s", path.Join(ctx.dir, ctx.output+".mp4"))
}

// Parse 解析出M3U8
func Parse(body string, ctx *Context) M3U8 {
	base, err := url.Parse(ctx.URL)
	if err != nil {
		log.Fatalf("解析URL失败: %v", err)
	}
	base.Path = path.Dir(base.Path)
	baseUrlPrefix := base.String()
	host := base.Host

	// 解析帧
	frame := ParseFrame(body, host, baseUrlPrefix)

	// 解析加密信息
	encrypt := ParseEncrypt(body, baseUrlPrefix)
	return M3U8{
		baseUrl:       ctx.URL,
		baseUrlPrefix: baseUrlPrefix,
		frames:        frame,
		encrypt:       encrypt,
		host:          host,
	}
}
