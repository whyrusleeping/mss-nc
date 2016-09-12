package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

const ProtocolID = "/multistream/1.0.0"

var listen *bool = flag.Bool("l", false, "listen on a port")
var verbose *bool = flag.Bool("v", false, "verbose output")

func main() {
	flag.Parse()

	if len(flag.Args()) < 2 {
		fmt.Printf("usage: %s <host> <port>\n", os.Args[0])
		return
	}

	if *listen {
		list, err := net.Listen("tcp", ":"+flag.Args()[1])
		if err != nil {
			fmt.Println("error: ", err)
			return
		}

		con, err := list.Accept()
		if err != nil {
			fmt.Println("error: ", err)
			return
		}

		fmt.Println("GOT CONN")

		doNC(con)
		return
	}

	con, err := net.Dial("tcp", fmt.Sprintf("%s:%s", flag.Args()[0], flag.Args()[1]))
	if err != nil {
		fmt.Println("error: ", err)
		return
	}

	doNC(con)
}

func OutPrintf(f string, a ...interface{}) {
	if *verbose {
		fmt.Printf("> "+f, a...)
	}
}
func InPrintf(f string, a ...interface{}) {
	if *verbose {
		fmt.Printf("< "+f, a...)
	} else {
		fmt.Printf(f, a...)
	}
}
func VPrintf(f string, a ...interface{}) {
	if *verbose {
		fmt.Printf(f, a...)
	}
}

func doNC(con net.Conn) {
	OutPrintf("%s\n", ProtocolID)
	err := writeDelimited(con, []byte(ProtocolID))
	if err != nil {
		fmt.Println("error: ", err)
		return
	}

	// Read server hello
	line, err := readDelimited(con)
	if err != nil {
		fmt.Println("error: ", err)
		return
	}

	InPrintf("%s\n", string(line))

	scan := bufio.NewScanner(os.Stdin)

	VPrintf("> ")
	for scan.Scan() {
		if scan.Err() != nil {
			fmt.Println("error: ", scan.Err())
			return
		}

		err := writeDelimited(con, scan.Bytes())
		if err != nil {
			fmt.Println("error: ", err)
			return
		}

		line, err := readDelimited(con)
		if err != nil {
			fmt.Println("error: ", err)
			return
		}

		switch scan.Text() {
		case "ls":
			// ls has a special response format
			// readd new line
			r := bytes.NewReader([]byte(string(line) + "\n"))
			br := &byteReader{r}
			count, err := binary.ReadUvarint(br)
			if err != nil {
				break
			}

			for i := uint64(0); i < count; i++ {
				p, err := readDelimited(r)
				if err != nil {
					break
				}

				fmt.Println(string(p))
			}
		default:
			InPrintf("%s\n", string(line))
			oldline := line
			line, err := readDelimited(con)

			if err == nil && strings.TrimSpace(string(line)) == ProtocolID {
				writeDelimited(con, []byte(ProtocolID))
				InPrintf("%s\n", ProtocolID)
				OutPrintf("%s\n", ProtocolID)
			} else if oldline[0] == '/' {
				writeDelimited(os.Stdout, line)
				go func() {
					_, err := io.Copy(os.Stdout, con)
					if err != nil {
						fmt.Printf("read error: %s", err)
					}
				}()
				_, err := io.Copy(con, os.Stdin)
				if err != nil {
					fmt.Printf("write error: %s", err)
				}
				return
			}
		}
		VPrintf("> ")
	}
}

// writeDelimited writes a varint-length-prefixed-newline-terminated message
func writeDelimited(w io.Writer, data []byte) error {
	buf := make([]byte, 8)
	n := binary.PutUvarint(buf, uint64(len(data)+1))
	_, err := w.Write(buf[:n])
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	if err != nil {
		return err
	}

	_, err = w.Write([]byte{'\n'})

	return err
}

// byteReader implements the ByteReader interface that ReadUVarint requires
type byteReader struct {
	io.Reader
}

func (br *byteReader) ReadByte() (byte, error) {
	var b [1]byte
	_, err := br.Read(b[:])

	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func readDelimited(r io.Reader) ([]byte, error) {
	br := &byteReader{r}
	length, err := binary.ReadUvarint(br)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, length)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}

	if len(buf) == 0 || buf[length-1] != '\n' {
		return nil, errors.New("message did not have trailing newline")
	}

	// slice off the trailing newline
	buf = buf[:length-1]

	return buf, nil
}
