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
	"github.com/pkg/sftp"
	// gossh "golang.org/x/crypto/ssh"
)

var (
	port               = "2222"
	pwd                = "123456"
	publicKey          []byte
	authorizedKeyBytes = []byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDJIQqtAyGTkrZzyIrsdwvJWx21h8XEgLozhyEnjmv5xLKDhlfvixsZZLENsAQ/aIjaU3irczyDU+WCacQaTTDjir/9F8hVIjXFPSCvX9BvGdEo+d6oVjCw/tOowHSMDxtDYzwoggqaS80jU8SGIgLWKR8Jy703rdBUUx6mRZKorvIlUiT8Tovd+87r5m+9dO00ndRzaWSX41fZ62Qxi37xrdSxH2V6gC87tUK9sBzM4n4wcu25ZjRZWIZNvbb3F3slQI8DAvvwy25H96najqeBoYYHTjCVuEHwKfcEfImajRB53GDfYg6X1ItBF85WC54mPjUjNBg0iBdUdZaRArY/")
	authorizedKey      = func() ssh.PublicKey {
		a, _, _, _, err := ssh.ParseAuthorizedKey(authorizedKeyBytes)
		if err != nil {
			log.Fatalln("cannot parse authorizedKeyBytes")
		}
		return a
	}()
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

// SftpHandler handler for SFTP subsystem
func SftpHandler(sess ssh.Session) {
	debugStream := io.Discard
	serverOptions := []sftp.ServerOption{
		sftp.WithDebug(debugStream),
	}
	server, err := sftp.NewServer(
		sess,
		serverOptions...,
	)
	if err != nil {
		log.Printf("sftp server init error: %s\n", err)
		return
	}
	if err := server.Serve(); err == io.EOF {
		server.Close()
		fmt.Println("sftp client exited session.")
	} else if err != nil {
		fmt.Println("sftp server completed with error:", err)
	}
}

func main() {
	publicKeyOption := ssh.PublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
		return ssh.KeysEqual(key, authorizedKey)
		// return true // allow all keys, or use ssh.KeysEqual() to compare against known keys
	})
	pwdOption := ssh.PasswordAuth(func(ctx ssh.Context, password string) bool {
		return password == pwd
	})
	ssh_server := ssh.Server{
		Addr: ":" + port,
		SubsystemHandlers: map[string]ssh.SubsystemHandler{
			"sftp": SftpHandler,
		},
	}
	ssh_server.SetOption(publicKeyOption)
	ssh_server.SetOption(pwdOption)
	ssh_server.Handle(func(s ssh.Session) {
		// publicKey = gossh.MarshalAuthorizedKey(s.PublicKey())
		// s.Write(publicKey)
		io.WriteString(s, fmt.Sprintf("Login user: %s\n", s.User()))
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

	// log.Println("starting ssh server on port " + port + "...")
	// log.Fatal(ssh_server.ListenAndServe())
	ssh_server.ListenAndServe()
}
