package emlfile

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	stdmail "net/mail"
	"net/textproto"
	"strings"
	"time"
)

var (
	UnDefinedName = "undefined"
	UnDefinedAddr = "undefined@undefined.net"
)

type EMLData struct {
	Size int // mail raw size

	Header      stdmail.Header     // mail headers
	From        *stdmail.Address   // obtained from Header, always be set
	Tos         []*stdmail.Address // obtained from Header
	Ccs         []*stdmail.Address // obtained from Header
	Subject     string             // obtained from Header
	Date        time.Time          // obtained from Header
	ContentType string             // obtained from Header
	Encoding    string             // obtained from Header

	RawBody []byte         // mail raw contents
	Parts   []*EMLBodyPart // parsed from RawBytes if Content-Type is multiparts/
}

type EMLBodyPart struct {
	Header textproto.MIMEHeader
	Body   []byte
}

// ParseEml parse the raw eml bytes into EMLData
func ParseEml(emlRaw []byte) (*EMLData, error) {
	var (
		data = &EMLData{Size: len(emlRaw)}
	)

	// parse raw eml
	msg, err := stdmail.ReadMessage(bytes.NewBuffer(emlRaw))
	if err != nil {
		return nil, fmt.Errorf("mail.ReadMessage() error: %v", err)
	}

	// process mail headers
	data.Header = msg.Header

	var (
		from     = msg.Header.Get("From")
		to       = msg.Header.Get("To")
		cc       = msg.Header.Get("Cc")
		subject  = msg.Header.Get("Subject")
		date     = msg.Header.Get("Date")
		ctype    = msg.Header.Get("Content-Type")
		encoding = msg.Header.Get("Content-Transfer-Encoding")
	)

	if subject == "" {
		return nil, errors.New("email should have a non-empty subject")
	}

	if from != "" {
		if data.From, err = parseMailAddressWithFallBack(from); err != nil {
			return nil, fmt.Errorf("parse mailAddr on [From] error: %v ", err)
		}
	} else {
		data.From = &stdmail.Address{Name: UnDefinedName, Address: UnDefinedAddr}
	}

	if to != "" {
		if data.Tos, err = parseMailAddressListWithFallBack(to); err != nil {
			return nil, fmt.Errorf("parse mailAddrList on [To] error: %v", err)
		}
	}

	if cc != "" {
		if data.Ccs, err = parseMailAddressListWithFallBack(cc); err != nil {
			return nil, fmt.Errorf("parse mailAddrList on [Cc] error: %v", err)
		}
	}

	data.Subject = decodeSubject(subject)

	if date != "" {
		if data.Date, err = stdmail.ParseDate(date); err != nil {
			return nil, fmt.Errorf("parse date on [Date] error: %v", err)
		}
	}

	data.ContentType = ctype
	data.Encoding = encoding

	// process mail body
	data.RawBody, err = ioutil.ReadAll(msg.Body)
	if err != nil {
		return nil, fmt.Errorf("ReadAll() on eml body error: %v", err)
	}

	// according by header Content-Type
	if !strings.HasPrefix(ctype, "multipart/") {
		return data, nil
	}

	data.Parts, err = parseMultiPartsBody(data.RawBody, ctype)
	if err != nil {
		return nil, fmt.Errorf("parseMultiPartsBody() error: %v", err)
	}

	return data, nil
}

func parseMultiPartsBody(body []byte, ctype string) ([]*EMLBodyPart, error) {
	// get boundary params
	_, params, err := mime.ParseMediaType(ctype)
	if err != nil {
		return nil, fmt.Errorf("mime.ParseMediaType() on [%s] error: %v", ctype, err)
	}

	// read each part of mail body
	var (
		bodyBuf         = bytes.NewBuffer(body)
		boundary        = params["boundary"]
		multiPartReader = multipart.NewReader(bodyBuf, boundary)
		ret             = make([]*EMLBodyPart, 0)
	)

	for {
		part, err := multiPartReader.NextPart()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("multipart.Reader().NextPart() error: %v", err)
		}

		partbs, err := ioutil.ReadAll(part)
		if err != nil {
			return nil, fmt.Errorf("multipart.Reader.ReadPart() error: %v", err)
		}

		ret = append(ret, &EMLBodyPart{Header: part.Header, Body: partbs})
	}

	return ret, nil
}

// parseMailAddressWithFallBack parse mail address with fall back ways (ignored encoded `Name`)
// if some charsets are not supported like GBK/windows-1252 with error messages like:
// missing word in phrase: charset not supported: \"gbk\" "
// See: https://github.com/golang/go/issues/7079
func parseMailAddressWithFallBack(addr string) (*stdmail.Address, error) {
	ret, err := stdmail.ParseAddress(addr)
	if err == nil {
		return ret, nil
	}

	if strings.Contains(err.Error(), "charset not supported") {
		if fields := strings.Fields(addr); len(fields) >= 2 {
			// directly split the original address and use the 2th field as parsed result
			return &stdmail.Address{Name: fields[1], Address: fields[1]}, nil
		}
	}

	return nil, fmt.Errorf("mail.ParseAddr() error: %v ", err)
}

func parseMailAddressListWithFallBack(addrs string) ([]*stdmail.Address, error) {
	ret := make([]*stdmail.Address, 0)

	for _, addr := range strings.Split(addrs, ",") {
		if addr == "" {
			continue
		}
		entry, err := parseMailAddressWithFallBack(addr)
		if err != nil {
			return nil, err
		}
		ret = append(ret, entry)
	}

	return ret, nil
}

func decodeSubject(subject string) string {
	var ret string
	for _, part := range strings.Fields(subject) {
		if ret == "" {
			ret = DecodeRFC2047(part)
		} else {
			ret += " " + DecodeRFC2047(part)
		}
	}
	return ret
}
