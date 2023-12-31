module github.com/sat1007/sshdgo

go 1.18

require (
	github.com/creack/pty v1.1.18
	github.com/gliderlabs/ssh v0.3.5
	github.com/pkg/sftp v1.13.6
	golang.org/x/crypto v0.13.0
)

replace github.com/creack/pty v1.1.18 => ../../photostorm/pty

replace github.com/gliderlabs/ssh v0.3.5 => ../ssh

require (
	github.com/anmitsu/go-shlex v0.0.0-20200514113438-38f4b401e2be // indirect
	github.com/kr/fs v0.1.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
)
