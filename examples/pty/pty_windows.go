//go:build windows
// +build windows

package main

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"runtime"

	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
)

func setWinsize(f pty.Pty, w, h int) {
	err := pty.Setsize(f, &pty.Winsize{
		Cols: uint16(w),
		Rows: uint16(h),
	})
	if err != nil {
		log.Println(err)
	}
}

func main() {
	ssh.Handle(func(s ssh.Session) {
		cmd := exec.Command("bash")
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd.exe")
		}
		ptyReq, winCh, isPty := s.Pty()
		if isPty {
			cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
			f, err := pty.Start(cmd)
			if err != nil {
				panic(err)
			}
			go func() {
				for win := range winCh {
					setWinsize(f, win.Width, win.Height)
				}
			}()
			go func() {
				io.Copy(f, s) // stdin
			}()
			go func() {
				io.Copy(s, f) // stdout
				s.Close()
			}()
			cmd.Wait()
		} else {
			io.WriteString(s, "No PTY requested.\n")
			s.Exit(1)
		}
	})

	log.Println("starting ssh server on port 2222...")
	log.Fatal(ssh.ListenAndServe(":2222", nil))
}
