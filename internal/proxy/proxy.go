package proxy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"nms-ring/internal/ring"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type Proxy struct {
	cmd *exec.Cmd
	ctx context.Context

	input  io.WriteCloser
	output io.Reader
}

var (
	colorRegex = regexp.MustCompile(`\[(\?\?|\d{2})\]\s*\((\d{1,3}),(\d{1,3}),(\d{1,3})\)`)
	levelDict  = map[string]string{
		"[SSS] ": "\x1b[48;2;255;215;0m SSS \x1b[0m ",
		"[SS+] ": "\x1b[48;2;255;69;0m SS+ \x1b[0m ",
		"[SS] ":  "\x1b[48;2;255;69;0m SS  \x1b[0m ",
		"[S] ":   "\x1b[48;2;255;140;0m  S  \x1b[0m ",
		"[A] ":   "\x1b[48;2;131;90;170m  A  \x1b[0m ",
		"[B] ":   "\x1b[48;2;70;130;180m  B  \x1b[0m ",
		"[C] ":   "\x1b[48;2;60;130;80m  C  \x1b[0m ",
		"[D] ":   "\x1b[48;2;245;245;245m  D  \x1b[0m ",
		"[E] ":   "\x1b[48;2;200;200;210m  E  \x1b[0m ",
	}
)

func (p *Proxy) Run() {
	err := p.cmd.Start()
	if err != nil {
		fmt.Println("启动程序失败:", err)
		return
	}

	go func() {
		// 循环获取输出信息
		msgCh := msgRead(p.output)

		for {
			select {
			case msg := <-msgCh:
				if msg.e != nil {
					if !errors.Is(msg.e, io.EOF) {
						fmt.Println("读取输出失败:", msg.e)
					}
					return
				}
				reader := transform.NewReader(bytes.NewReader(msg.m), simplifiedchinese.GBK.NewDecoder())
				utf8text, err := io.ReadAll(reader)
				if err != nil {
					fmt.Println("转换编码失败:", err)
					return
				}

				text := string(utf8text)
				// 判断星球类型是否播放铃声
				if strings.Contains(text, "[S]") || strings.Contains(text, "[SS]") || strings.Contains(text, "[SS+]") || strings.Contains(text, "[SSS]") {
					go ring.Play()
				}
				for key, val := range levelDict {
					text = strings.ReplaceAll(text, key, val)
				}

				// 处理颜色输出
				list := colorRegex.FindAllStringSubmatch(text, -1)
				for _, item := range list {
					if item[1] == "??" {
						item[1] = "  "
					} else {
						item[1] = fmt.Sprintf(" %s ", item[1])
					}
					text = strings.ReplaceAll(text, item[0], fmt.Sprintf("\x1b[48;2;%s;%s;%sm%s\x1b[0m", item[2], item[3], item[4], item[1]))
				}

				// 显示输出
				fmt.Print(text)
				// 判断是否要用户输入确认
				if strings.Contains(text, "[Y]我同意 [N]不同意:") || strings.Contains(text, "请输入命令:") || strings.Contains(text, "请输入选择:") {
					// 将用户输入转发
					var input string
					fmt.Scan(&input)
					input += "\n"

					_, err := p.input.Write([]byte(input))
					if err != nil {
						fmt.Println("写入用户输入失败:", err)
						return
					}
				}

			case <-p.ctx.Done():
				return
			}
		}
	}()

	// 等待程序退出
	err = p.cmd.Wait()
	if err != nil {
		fmt.Println("程序异常退出:", err)
		return
	}
}

type msg struct {
	m []byte
	e error
}

func msgRead(p io.Reader) <-chan msg {
	ch := make(chan msg, 10)
	go func() {
		for {
			buf := make([]byte, 1<<10)
			length, err := p.Read(buf)
			if err != nil {
				ch <- msg{m: buf[:0], e: err}
				close(ch)
				return
			}
			ch <- msg{m: buf[:length]}
		}
	}()
	return ch
}

func Run(programPath string) {
	if len(programPath) == 0 {
		fmt.Println("程序路径不能为空")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := Proxy{
		ctx: ctx,
		cmd: exec.CommandContext(ctx, programPath),
	}

	// 设置管道
	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		fmt.Println("创建标准输出管道失败:", err)
		return
	}
	defer stdout.Close()

	stderr, err := p.cmd.StderrPipe()
	if err != nil {
		fmt.Println("创建标准错误管道失败:", err)
		return
	}
	defer stderr.Close()
	p.output = io.MultiReader(stdout, stderr)

	p.input, err = p.cmd.StdinPipe()
	if err != nil {
		fmt.Println("创建输入管道失败:", err)
		return
	}
	defer p.input.Close()

	p.Run()
}
