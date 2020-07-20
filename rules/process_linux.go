package rules

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Dreamacro/clash/common/cache"
	C "github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/log"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var processCache = cache.New(time.Minute)

type Process struct {
	adapter string
	process string
}

func (p *Process) RuleType() C.RuleType {
	return C.Process
}

func (p *Process) Match(metadata *C.Metadata) bool {
	key := fmt.Sprintf("%s:%s:%s", metadata.NetWork.String(), metadata.SrcIP.String(), metadata.SrcPort)
	cached := processCache.Get(key)
	if cached == nil {
		port, _ := strconv.Atoi(metadata.SrcPort)

		processName, err := queryProcessNameByAddr(metadata.NetWork, metadata.SrcIP, port)
		if err != nil {
			log.Debugln("[%s] Resolve process of %s failure: %s", C.Process.String(), key, err.Error())
		}

		processCache.Put(key, processName, time.Second*1)

		cached = processName
	}

	return strings.EqualFold(cached.(string), p.process)
}

func (p *Process) Adapter() string {
	return p.adapter
}

func (p *Process) Payload() string {
	return p.process
}

func (p *Process) NoResolveIP() bool {
	return true
}

func NewProcess(process string, adapter string) (*Process, error) {
	return &Process{
		adapter: adapter,
		process: process,
	}, nil
}

const pathTcp = "/proc/net/tcp"
const pathUdp = "/proc/net/udp"
const pathTcp6 = "/proc/net/tcp6"
const pathUdp6 = "/proc/net/udp6"
const pathProc = "/proc"
const indexOfLocalAddr = 1
const indexOfInode = 9
const indexOfUid = 7

var ErrInvalidIP = errors.New("invalid ip")

func queryProcessNameByAddr(network C.NetWork, sourceAddress net.IP, sourcePort int) (string, error) {
	println(time.Now().Nanosecond())
	defer println(time.Now().Nanosecond())

	inode, _, err := queryInodeUidByAddr(network, sourceAddress.To4(), sourcePort)
	if err != nil {
		inode, _, err = queryInodeUidByAddr(network, sourceAddress.To16(), sourcePort)
		if err != nil {
			return "", err
		}
	}

	return queryCmdlineByInode(inode)
}

func queryInodeUidByAddr(network C.NetWork, sourceAddress net.IP, sourcePort int) (int, int, error) {
	if sourceAddress == nil {
		return 0, 0, ErrInvalidIP
	}

	var port [2]byte

	var path string
	var address []byte

	if len(sourceAddress) == 16 {
		if network == C.TCP {
			path = pathTcp6
		} else {
			path = pathUdp6
		}

		address = []byte{
			sourceAddress[15], sourceAddress[14], sourceAddress[13], sourceAddress[12],
			sourceAddress[11], sourceAddress[10], sourceAddress[9], sourceAddress[8],
			sourceAddress[7], sourceAddress[6], sourceAddress[5], sourceAddress[4],
			sourceAddress[3], sourceAddress[2], sourceAddress[1], sourceAddress[0]}
	} else {
		if network == C.TCP {
			path = pathTcp
		} else {
			path = pathUdp
		}

		address = []byte{sourceAddress[3], sourceAddress[2], sourceAddress[1], sourceAddress[0]}
	}

	binary.BigEndian.PutUint16(port[:], uint16(sourcePort))

	return queryInodeUid(path, address, port[:])
}

func queryInodeUid(path string, ip []byte, port []byte) (int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return -1, -1, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	local := fmt.Sprintf("%s:%s", hex.EncodeToString(ip), hex.EncodeToString(port[:]))

	for {
		row, _, err := reader.ReadLine()
		if err != nil {
			return -1, -1, err
		}

		items := strings.Fields(string(row))

		if strings.EqualFold(local, items[indexOfLocalAddr]) {
			inode, err := strconv.Atoi(items[indexOfInode])
			if err != nil {
				return -1, -1, err
			}

			uid, err := strconv.Atoi(items[indexOfUid])
			if err != nil {
				return -1, -1, err
			}

			return inode, uid, nil
		}
	}
}

func queryCmdlineByInode(inode int) (string, error) {
	files, err := ioutil.ReadDir(pathProc)
	if err != nil {
		return "", err
	}

	buffer := make([]byte, syscall.PathMax)
	socket := []byte(fmt.Sprintf("socket:[%d]", inode))

	for _, f := range files {
		processPath := path.Join(pathProc, f.Name())
		fdPath := path.Join(processPath, "fd")

		fds, err := ioutil.ReadDir(fdPath)
		if err != nil {
			continue
		}

		for _, fd := range fds {
			n, err := syscall.Readlink(path.Join(fdPath, fd.Name()), buffer)
			if err != nil {
				continue
			}

			if bytes.Compare(buffer[:n], socket) == 0 {
				cmdline, err := ioutil.ReadFile(path.Join(processPath, "cmdline"))
				if err != nil {
					return "", err
				}

				return splitCmdline(cmdline), nil
			}
		}
	}

	return "", syscall.EEXIST
}

func splitCmdline(cmdline []byte) string {
	indexOfEndOfString := len(cmdline)

	for i, c := range cmdline {
		if c == 0 {
			indexOfEndOfString = i
			break
		}
	}

	return filepath.Base(string(cmdline[:indexOfEndOfString]))
}
