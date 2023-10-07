package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var signer = func() ssh.Signer {
	pkeybytes := []byte(`-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAABFwAAAAdzc2gtcn
NhAAAAAwEAAQAAAQEAySEKrQMhk5K2c8iK7HcLyVsdtYfFxIC6M4chJ45r+cSyg4ZX74sb
GWSxDbAEP2iI2lN4q3M8g1PlgmnEGk0w44q//RfIVSI1xT0gr1/QbxnRKPneqFYwsP7TqM
B0jA8bQ2M8KIIKmkvNI1PEhiIC1ikfCcu9N63QVFMepkWSqK7yJVIk/E6L3fvO6+ZvvXTt
NJ3Uc2lkl+NX2etkMYt+8a3UsR9leoAvO7VCvbAczOJ+MHLtuWY0WViGTb229xd7JUCPAw
L78MtuR/ep2o6ngaGGB04wlbhB8Cn3BHyJmo0Qedxg32IOl9SLQRfOVgueJj41IzQYNIgX
VHWWkQK2PwAAA8jvsjmm77I5pgAAAAdzc2gtcnNhAAABAQDJIQqtAyGTkrZzyIrsdwvJWx
21h8XEgLozhyEnjmv5xLKDhlfvixsZZLENsAQ/aIjaU3irczyDU+WCacQaTTDjir/9F8hV
IjXFPSCvX9BvGdEo+d6oVjCw/tOowHSMDxtDYzwoggqaS80jU8SGIgLWKR8Jy703rdBUUx
6mRZKorvIlUiT8Tovd+87r5m+9dO00ndRzaWSX41fZ62Qxi37xrdSxH2V6gC87tUK9sBzM
4n4wcu25ZjRZWIZNvbb3F3slQI8DAvvwy25H96najqeBoYYHTjCVuEHwKfcEfImajRB53G
DfYg6X1ItBF85WC54mPjUjNBg0iBdUdZaRArY/AAAAAwEAAQAAAQB6dPGpEUT6MtN/f1SG
UJ0Ohbl68yWIVNAJ23ZmPSKkugvuZHdZ05o2RcY/DTIo4R6hvzyzNsBbPVN5qafKU8E2aR
4nnLlOjDus0WD7Jh0j59YfWrMkTwXqXdzE3BiZxgDVcLAKAdMuyoQlxDTdbgvIFNVfA3s7
UUqMbOc2WRnINVcuSQ8UGj9vFnCs97ymxda9e1RGjLZhgU6ap/O65qZSWeSpFmUGmZE2gm
p2WKeiygrQix3DQ2WNAM4aLcW7u289nGol3yAkrMO8b/iEGOUFbVETQLwEhx7t0XL4pi3z
44RKpEYe2Y836mNFxqkGsNgVCeXnl3/j59blEPclzGVBAAAAgAf6x+HWkaz4dasCh9z+2G
qWJLpsSVUzDU1PnYo/abNyrQrUgK4y49C07Xx0f5M7A425JBaZ3ljzTKSDvDeOxelFVnKw
RMK+6yLKdLSvRBpfR26/ccryrhg1kVzA0bOyRIbRpF4A9kkLWBIInAqZzSGB4fpjD3DMf+
92XiBsjhFZAAAAgQDyoRTOioa1jJJELXm1kj5oyBbXAuTgixA7CjMZVIVw7jmTRFvDzejS
L6Es9Z3txyjUPBEXgt+VX5DC0hQa9xPlK9DG1+plVy/X3K1VT80vKBSckPVd5SCK8ievZ0
E+ytut8krxiUsJVyTf/DBSTVfEAX7o1vYqk3lxKj/xtpDq0QAAAIEA1DZ+DPoec81xFp+G
IssGfgnJ+L52WAy5WKhPnHCvddoVjZYCrzsKxUbFlsVlGb0TYkL5pL7QXvG7lFjM4jDyE9
gTvlm3okPcvjOhJEHC8EU134OnhdwinlzX+CyTuP6RzjkJPLWE9beZAEbAnOmMdGQYn9wC
V6VygtJAPclKtA8AAAAQdmFncmFudEBkZWJpYW4xMQECAw==
-----END OPENSSH PRIVATE KEY-----`)
	signer, err := ssh.ParsePrivateKey(pkeybytes)
	if err != nil {
		log.Fatal(err)
	}
	return signer
}()

type Cli struct {
	user       string
	pwd        string
	ip         string
	port       string
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

func NewSSHClient(user, pwd, ip, port string) Cli {
	return Cli{
		user: user,
		pwd:  pwd,
		ip:   ip,
		port: port,
	}
}

func (c *Cli) getConfig(signer ssh.Signer) *ssh.ClientConfig {
	config := &ssh.ClientConfig{
		User: c.user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
			ssh.Password(c.pwd),
		},
		Timeout:         30 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return config
}
func (c *Cli) Connect() error {
	config := c.getConfig(signer)
	client, err := ssh.Dial("tcp", c.ip+":"+c.port, config)
	if err != nil {
		return fmt.Errorf("connect server error: %w", err)
	}
	sftp, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("new sftp client error: %w", err)
	}
	c.sshClient = client
	c.sftpClient = sftp
	return nil
}

func (c Cli) UploadFile(localFile, remoteFileName string) (int, error) {
	if c.sshClient == nil {
		if err := c.Connect(); err != nil {
			return -1, err
		}
	}
	file, err := os.Open(localFile)
	if nil != err {
		return -1, fmt.Errorf("open local file failed: %w", err)
	}
	defer file.Close()
	ftpFile, err := c.sftpClient.Create(remoteFileName)
	if nil != err {
		return -1, fmt.Errorf("Create remote path failed: %w", err)
	}
	defer ftpFile.Close()
	fileByte, err := io.ReadAll(file)
	if nil != err {
		return -1, fmt.Errorf("read local file failed: %w", err)
	}
	ftpFile.Write(fileByte)
	return 0, nil
}

func main() {
	foobarPtr := flag.Bool("foobar", false, "")
	username := "demo"
	pwdPtr := flag.String("pwd", "123456", "")
	ipPtr := flag.String("ip", "127.0.0.1", "")
	portPtr := flag.String("port", "2222", "")
	srcPtr := flag.String("src", "./README.md", "")
	dstPtr := flag.String("dst", "/tmp/README.md", "")
	flag.Parse()
	if !*foobarPtr {
		return
	}
	client := NewSSHClient(username, *pwdPtr, *ipPtr, *portPtr)
	_, err := client.UploadFile(*srcPtr, *dstPtr)
	if err != nil {
		fmt.Printf("upload failed: %v\n", err)
		return
	}
}
