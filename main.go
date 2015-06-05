package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
)

func writeDelimited(w io.Writer, data []byte) error {
	buf := make([]byte, 8)
	n := binary.PutUvarint(buf, uint64(len(data)))
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

	nline, err := br.ReadByte()
	if err != nil {
		return nil, err
	}

	if nline != '\n' {
		panic("oh my god oh my god oh my god oh my god")
	}

	return buf, nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("usage: %s <host> <port>\n")
		return
	}

	con, err := net.Dial("tcp", fmt.Sprintf("%s:%s", os.Args[1], os.Args[2]))
	if err != nil {
		fmt.Println("error: ", err)
		return
	}

	line, err := readDelimited(con)
	if err != nil {
		fmt.Println("error: ", err)
		return
	}

	fmt.Println(string(line))

	scan := bufio.NewScanner(os.Stdin)
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
			r := bytes.NewReader(line)
			for {
				p, err := readDelimited(r)
				if err != nil {
					break
				}

				fmt.Println(string(p))
			}
		default:
			fmt.Println(string(line))
			if line[0] == '/' {
				go io.Copy(os.Stdout, con)
				io.Copy(con, os.Stdin)
				return
			}
		}
	}
}
