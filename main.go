package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

func main() {
	pathImages := "images"
	if _, err := os.Stat(pathImages); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(pathImages, os.ModePerm)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	pathEmails := "emails"
	if _, err := os.Stat(pathEmails); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(pathEmails, os.ModePerm)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	fmt.Println("connecting to server")
	// yandex address
	c, err := client.DialTLS("imap.yandex.com:993", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("connected to server")
	defer c.Logout()
	if err := c.Login("hoanglh1311", "wpuwokrdwddsnbco"); err != nil {
		fmt.Println(err)
		return
	}

	mbox, err := c.Select("INBOX", false)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Get the last message
	if mbox.Messages == 0 {
		fmt.Println("No message in mailbox")
		return
	}
	from := uint32(1)
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, mbox.Messages)
	// seqset.AddNum(mbox.Messages)
	messages := make(chan *imap.Message, mbox.Messages)
	var section imap.BodySectionName
	items := []imap.FetchItem{section.FetchItem()}
	go func() {
		if err := c.Fetch(seqset, items, messages); err != nil {
			fmt.Println(err)
			return
		}
	}()
	for msg := range messages {
		if msg == nil {
			fmt.Println("Server didn't returned message")
			return
		}

		r := msg.GetBody(&section)
		if r == nil {
			fmt.Println("Server didn't returned message body")
			return
		}

		// Create a new mail reader
		mr, err := mail.CreateReader(r)
		if err != nil {
			fmt.Println(err)
			return
		}
		// Print some info about the message
		header := mr.Header
		if subject, err := header.Subject(); err == nil {
			fmt.Println("Subject:", subject)
		}

		p, err := mr.NextPart()
		if err != nil {
			fmt.Println(err)
			return
		}
		b, _ := ioutil.ReadAll(p.Body)
		fmt.Println("Body:", string(b))
		name := strings.ReplaceAll(header.Get("Subject"), " ", "_")
		newPath := filepath.Join("emails", name+".txt")
		if _, err := os.Stat(newPath); errors.Is(err, os.ErrNotExist) {
			f, err := os.Create(newPath)
			if err != nil {
				fmt.Println(err)
				return
			}
			defer f.Close()
			_, err2 := f.WriteString(string(b))
			if err2 != nil {
				fmt.Println(err2)
				return
			}
		}
		// try to save image
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Fatal(err)
			}

			switch h := p.Header.(type) {
			case *mail.AttachmentHeader:
				// This is an attachment
				filename, _ := h.Filename()
				subject := header.Get("Subject")
				log.Printf("Got attachment: %v\n", filename)
				// Create file with attachment name
				filename = filepath.Join("images", subject+"_"+filename)
				fmt.Println(filename)
				file, err := os.Create(filename)
				if err != nil {
					log.Fatal(err)
				}
				// using io.Copy instead of io.ReadAll to avoid insufficient memory issues
				size, err := io.Copy(file, p.Body)
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("Saved %v bytes into %v\n", size, filename)
			}
		}
	}

}
