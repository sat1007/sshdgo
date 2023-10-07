//go:build !windows
// +build !windows

package main

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

var publicKey []byte
var authorizedKeyBytes = []byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDJIQqtAyGTkrZzyIrsdwvJWx21h8XEgLozhyEnjmv5xLKDhlfvixsZZLENsAQ/aIjaU3irczyDU+WCacQaTTDjir/9F8hVIjXFPSCvX9BvGdEo+d6oVjCw/tOowHSMDxtDYzwoggqaS80jU8SGIgLWKR8Jy703rdBUUx6mRZKorvIlUiT8Tovd+87r5m+9dO00ndRzaWSX41fZ62Qxi37xrdSxH2V6gC87tUK9sBzM4n4wcu25ZjRZWIZNvbb3F3slQI8DAvvwy25H96najqeBoYYHTjCVuEHwKfcEfImajRB53GDfYg6X1ItBF85WC54mPjUjNBg0iBdUdZaRArY/")
var authorizedKey = func() ssh.PublicKey {
	a, _, _, _, err := ssh.ParseAuthorizedKey(authorizedKeyBytes)
	if err != nil {
		log.Fatalln("cannot parse authorizedKeyBytes")
	}
	return a
}()

func setWinsize(f pty.Pty, w, h int) {
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
}

func main() {
	publicKeyOption := ssh.PublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
		return ssh.KeysEqual(key, authorizedKey)
		// return true // allow all keys, or use ssh.KeysEqual() to compare against known keys
	})
	ssh.Handle(func(s ssh.Session) {
		publicKey = gossh.MarshalAuthorizedKey(s.PublicKey())
		io.WriteString(s, fmt.Sprintf("public key used by %s:\n", s.User()))
		s.Write(publicKey)
		cmd := exec.Command("bash")
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
			io.Copy(s, f) // stdout
			cmd.Wait()
		} else {
			io.WriteString(s, "No PTY requested.\n")
			s.Exit(1)
		}
	})

	log.Println("starting ssh server on port 2222...")
	log.Fatal(ssh.ListenAndServe(":2222", nil, publicKeyOption))
}
