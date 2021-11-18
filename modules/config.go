package modules

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts" // module for parsing `known_hosts` file
)

var (
	knownHostFile string = path.Join(os.Getenv("HOME"), ".ssh/known_hosts")
	keyErr        *knownhosts.KeyError
)

func makeClientConfig(user, password string, timeout int) ssh.ClientConfig {
	var sshKey string = ""
	if proj == "ipvt" {
		sshKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAuCMpZDvoAmZ8nT0FJgsndepav6Qg6ATnaz81Pb2e4WC9Uh0Dn62Lb7riC8D6
NekIkQgBeA0s7VENqkFY2jfXWtNW2luEI+Q2UOTE5FP9EUsTCpf/VzXRKYDQ8DGavD8JCejGvRVg
g3bImLoeQHjALZWAAN2JEfg3iN7bVW1HpxHfTyk/V7QvZi2pFfNh7pmjrnQQ5eOp5gZo/K3ov9My
XXBDRg0TSHqkimj63dbnrCVoMUKYHHHBVfn/bg+eEGuA+HnEbnscnKP3CBAx4Ova7hmd14C6AuKG
qBZ+hEAI6MC+/98eFnfLm/1WIp7xr03a2NFr2aGB6jYJH8H8uCxrkQIDAQABAoIBAQCrz7MpYXRD
+RjBJlSNgM7bCUaPljdWBm1f9mRZLXr31GoSGhxte2KlZ/zO9ssATmW93XhxaenHrV9QwxSC2KPK
jXN23vlW+/NqW2sTmQKChkhIFDceSorVbOqHk+FLUI3Af0Ag4bdiMw1M5Cyh/4lhfyMmq47gA3jd
7wM8nHDFAyONYrPvEbxr5fqDn1jZt5iPUc1U1Ivxsd7K92mi9URk67exdSNhAEwyXN9LaaD9tJ0V
w2f7xIz2/zqMrr9Fg42WRT2Fs5TOwgPt6Zsk1y4QWmN3F2P1FywX+IQatjpqoKJ6c5GLaf3mlQ34
5q4QAEVrHX4vxL4iz78JQJKs0ckZAoGBAOu7PicHfRAscr0Xb9VyLVi+e0Co3uwuDpVHUV+MUp/Q
oWMsyrWPDyoFQz5tMMXnADNXOgRQrg5AFPjs3ZxT4wvX7SBEyK9C16Vfnp1ZZkgMp0rdzqwJX2ea
SM6ePl5brmsYWu33+p4VyhLA+CBmVO1TEBzve7RDzO95hwjpxPqrAoGBAMf4RBN91P6H8pukPyhG
d2tJSJzEjOfTHzxoUEiv0Y6ynpQEm7ngFkegjBqiKYI4Q+53yM1nluKNmbrwpjsFvee8ShP0CgNw
7cpvy33plVG95uRJ1j78wZHIR/bE/9PtFkq2/LLOjicIaS6bzGHqNP9CPbCe3Qsq0cDNAbDTc3Kz
AoGALKm4V+q6TlMtlhgXr0hHwTWt4o1cV0FOsAfoKgNLME52FXVKHuYxCFQg7nX/tK9UwWV4b5Ld
t4N6tcMjJdha/0Z0/hUqNNKkSwf35HYow+Pp2mx2GPBZrCZ2PveKd5RFUrM1nzrJuCQGulDncjQR
STngpqrVNE1YSdMru4uXL6sCgYA9U7t1Cts+sGzMJOQlsu6+3XvCcFkSY+IpgYhVsm4fSFJv7LXh
nILYtrkhFeiLAjx7LwtLS2Cv3GNwPIuOgGVMY6eIVQiZI5IZyo1q8S8VIZtlGev150hqMDU5zhLz
aLu0cEgxgj2AZQj0/V0CDnTwb91BhIcf/KLVcUn6c+7tFwKBgQDnzrgVyYXZAp2RgY60mvVTZK7k
dQFCakawJdOppWHuwc0tofjggZBSibdqfPKKE+GDY3nKMtPT0F4fJsJ3wh/edfgRDLtx+BiR5GRq
fHFKQ/ly5lOhsDTyz93eErLhBtFN0yJuZe/Ua0ITajG+DkP960F14Ext3iP9050l0/yiOQ==
-----END RSA PRIVATE KEY-----`
	} else if proj == "gwn" {
		sshKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAh+xq9qSBxorsxWjMjVcLy77P3Z37zYmKoW7aAVbtrpyQjtCy37Ot8gQzrXqv
O5MBI8MLXUVgQjtXf3GHJXh6QiI0g6ZMx1Y5P7NQ0SXgwS99Gi9AreUPfmDwD02V8WDkaoRxhvh9
yeW4JTIT8uIk31ANkRcOP026zG9TuXRRCb91SmJWY3dUmsl+FtPh0XrcUwCaiKQpqpi+S5KDSoPp
7mEdpKDRiqHR+jH1i1blG0t+N9PTqD6hXuPwkIDVIhY5exoAW65ynbIJU2J9oiWzUyvRyII3SdeK
SeFkfGMc6Mo1fJQZEuL4WDWG79k8TsGlatUvL9CpubhvP2Q+sq0GVQIDAQABAoIBAHIFqKEo1Vj5
d7AOvvGeYN3VPsi5W99LD0lnFWhkRNTir/2uIy+3qibI0ZUowtEl+6HFX8YpiZtl7nuRf/6191F9
IjKCEgxyT2oZgaVBsi49KSQLGaYG7p1ksl0UB8HKNzMH0biYweTZWUWSodtxS98tZ4QcJC6EhTwz
87cyVTzn7AtuMhW40XNtSC2Ix6enpW45Lbauj7uOq9y+UlbvwDs8HdzSZ2l9eXix+XkSRh/lanfv
O7vNqLFNZ4BEnF/k6X0k/QVyQsgoJPSjLIOEJKnoHu805JTLJkxHRU3egK4+kkRVNWpGhExtZnsH
KlsR/4IXywPy68MpNrDb6ndvxqECgYEA2HfosM5HDrot8SANGD9eaSS1S5xV09n80f/vTTCoqGOC
0DmciZ1mr+uGU55KerKxv5Zzg55NaKC9H6PCqd1h54m2ix5OovBa+mm310lQBU4uWvxv86aor3f4
NZYIJNRyClET96v4iKPY5PBjxk+cSp9fnPexIFdakbsBlldQAZ0CgYEAoL72nrlhGHFdgTUJ5olp
UaQGrmLwLKHetrwLXz1q7To3XxEBZvQRNyjky3r0z5ytH47shp4PDuLATxpw7vW0pO5AoZwEI0cg
MLXp6rcPd0vQ/Gi2P4ztHl6lZCov9+wdpLk0ZgD9eCuY70/yrUg39scMVEsyTIv6KUJhpUqQ9hkC
gYB2Ojd6qY8hugnxkkHUpneYEZZ7L22pAXwV7iTzEcpr/b2qoGygtLkrCAYbkW64SPTP4QpCLqm0
pWXl6/kb8W/Rl+N9ZBYq4/+smSTxjncIDsU+qHU0rCehfnKwQbs88lM+0z2GngRmDKcfkzPLUhvY
necIwjeZpFwxD+Q0CogF+QKBgBSmU4IyaFG+hneJu7rc3iW5v435cccaCEVLRN64Qhs1hlV1FswG
AIwoebPWGMru4qKSNrpYZbDQ2kmPKQDZmZoybSUVqMZrZaw8Qk9/ssHtgGxce0UQsolbOT3z2XOs
5mX50MxlxioA18WubItsIpxWF7RgcJa6yKkyON8GBDdJAoGAWfbRenHj8RG4gy+oBlzUIx3QPMeM
LScoY+DneGCBb22XH9K9XJAAr1pcAtirC35DG7R1pxKKKHBDQwxa4WXHetn43SRLxH2hu54GMuWv
ror6M8kNT9AIg+2Ih+9OZ2Yg7+dZs6EPNm/KMC9Mkk8TT8cg7ZHjG/+yzaxN9Sc+Xpg=
-----END RSA PRIVATE KEY-----`
	} else if proj == "gdms" {
		sshKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAlON23U9Wa6FPdhn5LfN99sZHgE62gZ3JMNdbrgYi2nln4l6QlJazfN60c3it
KTMmCkLXaKN5O/f1SHfPdFV1NiYQYNK/jr+lvq5k33NWNJ+Jfs//rMMTM7UCuIwzzK/Ux7trJqzg
uJYWjVXSv+fsRqNERrqUucVPR/2y7P9hwV3pNZYsVzieqAYH+1yLM+uaQRQ8wDNWWrHPDimj9HTG
p0BPKbTpYEVX1bemrIKqCh6VoELPmglXCB7+ky8xSABZ2hXJLYx3f3WvaL8LLHbwKx7aY5b9pSUX
+DsLmrDACqlf+ay0lt7FhIa5kqeJRn1c9c7Sk+mQnR4iJHdQrsg9RwIDAQABAoIBAHL4Xv4wWIyO
KN0K23bXUoqvu+DhZ18Zp5V2h04PYPSR0V1lJoIHCRzAoFmWcyigXZyI1uvvaBudsqO+GM09fGBO
7lDGDLO/W86tvikqWAQUucjZ9xXCIV7JRQubABjsIgQFvo1D69e91jZXcoFMJxC43G09gMEfEsSF
rJgD8Mmueiui4u4i5Nun2BK5t6Zj2mTrs+H9HIiZQx9NOGO6OgMLOYRwK8YeqCjHLDqXs3FZlB2D
LI8J8AHlUSl3Zzhdy8bWA5jHHtjOtHU+22xuxUcvcnQyEEn1B3KWRlmyVyQdNMl7nb+1sKSEcIMu
ggOEgVFwIG6SjTl2LxQodh9xb+ECgYEA3NK3R7UUG2zlZr40VksozGw50J1ww+Hgge+3i+vX/aWV
1U8jE9S4nNAWLHFRwDq0SpHIkAjZcSGwWT8CUDObBnhMd0JNntTlSkxqpD1IIZ3MY8+hnedXVT51
1jKba38EVeactBhCTcpK7Aru8iqpGHjUInRH2+5j+KyEgTQXUvcCgYEArJs3z2MeWe4L00Kpb8FX
hccvNd0eVRJ/oG5YJ64cddEP0m0jh9LQHVapocKdajEfKm2eNZ7xpuGN2M5c115L8OSjUp+FFOoX
kbj8zypBjOWXQHAMzYxQYF0EeL3ZI5Dg6c6rGNrnJqHXNnEwo8PiCvuB6jKbRGmCKm2OLwBGhDEC
gYEApq324g1wi/L6hT70yl4ZAZR/X1Q3leOdTvbemKMAedjO7B+73nAmmVcIJauzhWHKTK8VsIK9
foNrVTIWUOtjDNMpBKvrYwRXvmlH5YjMNXOin4RN/Z5tcU6gK3ovjkhlwWE8z0OwaH9VZi4qhmhl
Eq/Bj/AtfXjHxSCTM+NZ56kCgYALwnN19KnPtLEnjoYesAx3d8+Wmt8DKsR5OKtW4LzdpgWu73KZ
QVqedRYPiEPTRU97Q4ag3phWJ03TtJOMtMb1vY4HBgk6GIzMh87pilZ28/lvEXM92c5sEkpIs56E
ls1MAKGViuxurF4OPn3y2lifKO17+ECt04Zjv50NRLaJMQKBgQDA4TAaSg63xhYM6MddMtKlm5R3
IForbSSEjPlTDV0HFwCLpbO83YIqsE387qewUs2N8bscH4cta5WNvHhqA3i6KwMDCkkwbjPtk+gC
E9HUoUQVI8Lpt5uYh6ywDCD+KWw9xFbhZocnIB2Bvcy1vsEcrQb/8IHHKd2l1W4iJu3akQ==
-----END RSA PRIVATE KEY-----`
	} else {
		var buf []byte
		homePath := os.Getenv("HOME")
		autoKey := path.Join(homePath, ".ssh/auto_generation.pem")
		idRSAKey := path.Join(homePath, ".ssh/id_rsa")

		buf, err := ioutil.ReadFile(autoKey)
		if err != nil {
			fmt.Printf("Authentication failed with %s, continue to authenticate with keyfile `id_rsa`\n", autoKey)
			if buf, err = ioutil.ReadFile(idRSAKey); err != nil {
				fmt.Printf("Authentication failed Again with %s, provide `password` parameter\n", idRSAKey)
			}
		}
		sshKey = string(buf)
	}

	buf := []byte(sshKey)

	var auth []ssh.AuthMethod
	key, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		log.Printf("parse key file failed with error %s used passwd only", err)
		auth = append(auth, ssh.Password(password))
	} else {
		auth = append(auth, ssh.Password(password))
		auth = append(auth, ssh.PublicKeys(key))
	}

	hostKeyCheck := ssh.InsecureIgnoreHostKey() // windows system without sshHostKey
	if runtime.GOOS != "windows" {
		hostKeyCheck = ssh.HostKeyCallback(func(host string, remote net.Addr, pubKey ssh.PublicKey) error {
			nhkc := newHostKeyCallback()
			err := nhkc(host, remote, pubKey)
			if errors.As(err, &keyErr) && len(keyErr.Want) > 0 {
				log.Printf("WARNING: %v is not a key of %s, either a MiTM attack or %s has reconfigured the host pub key.", hostKeyString(pubKey), host, host)
				return keyErr
			} else if errors.As(err, &keyErr) && len(keyErr.Want) == 0 {
				log.Printf("WARNING: %s is not trusted, adding this key: %q to known_hosts file.", host, hostKeyString(pubKey))
				addToKnownHostsFile(host, remote, pubKey)
				return nil
			}
			// log.Printf("Pub key %s exists for %s.", hostKeyString(pubKey), host)
			log.Printf("Pub key exists for %s.", host)
			return nil

		})
	}
	config := ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: hostKeyCheck,
		Timeout:         time.Duration(timeout) * time.Second,
	}
	return config
}

// pase multiple ssh_keys in local file, eg: `~/.ssh/id_rsa`
func parseKeys() []byte {
	// NOT used in remoteSsh
	// auto_generation.pem
	homePath := os.Getenv("HOME")

	buf, err := ioutil.ReadFile(homePath + "/.ssh/id_rsa")
	if err != nil {
		fmt.Println(err)
	}
	var sshKeys []byte
	for _, n := range strings.SplitAfter(string(buf), "-----END RSA PRIVATE KEY-----") {
		s := []byte(n)
		sshKeys = append(sshKeys, s...) // Attention, three dots notation
	}

	return sshKeys
}

// create human-readable SSH-key strings defined in known_hosts file
func hostKeyString(k ssh.PublicKey) string {
	return k.Type() + " " + base64.StdEncoding.EncodeToString(k.Marshal()) // e.g. "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY...."
}

// create ssh.HostKeyCallback
func newHostKeyCallback() ssh.HostKeyCallback {
	hostKeyCallBack, err := knownhosts.New(knownHostFile)
	if err != nil {
		log.Fatal("an error occured during creating HostKeyCallback: ", err)
	}
	return hostKeyCallBack
}

// add remote host host publicKey to knonw_hosts file
func addToKnownHostsFile(host string, remote net.Addr, pubKey ssh.PublicKey) error {
	f, _ := os.OpenFile(knownHostFile, os.O_APPEND|os.O_WRONLY, 0644)
	defer f.Close()

	knownHostLine := knownhosts.Normalize(remote.String())
	hostItem := knownhosts.Line([]string{knownHostLine}, pubKey)

	// hash host value and write to `known_hosts` file
	hostItemSlice := strings.Split(hostItem, " ")
	hashHost := knownhosts.HashHostname(hostItemSlice[0])
	hostItemSlice[0] = hashHost
	_, err := f.WriteString(fmt.Sprintf("%s\n", strings.Join(hostItemSlice, " ")))
	return err
}
