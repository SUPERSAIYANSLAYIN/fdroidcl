// Copyright (c) 2015, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package fdroidcl

import (
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

type Index struct {
	Repo struct {
		Name        string `xml:"name,attr"`
		PubKey      string `xml:"pubkey,attr"`
		Timestamp   int    `xml:"timestamp,attr"`
		URL         string `xml:"url,attr"`
		Version     int    `xml:"version,attr"`
		MaxAge      int    `xml:"maxage,attr"`
		Description string `xml:"description"`
	} `xml:"repo"`
	Apps []App `xml:"application"`
}

type CommaList []string

func (cl *CommaList) FromString(s string) {
	*cl = strings.Split(s, ",")
}

func (cl *CommaList) String() string {
	return strings.Join(*cl, ",")
}

func (cl *CommaList) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var content string
	if err := d.DecodeElement(&content, &start); err != nil {
		return err
	}
	cl.FromString(content)
	return nil
}

func (cl *CommaList) UnmarshalText(text []byte) (err error) {
	cl.FromString(string(text))
	return nil
}

type HexVal []byte

func (hv *HexVal) FromString(s string) error {
	b, err := hex.DecodeString(s)
	if err != nil {
		return err
	}
	*hv = b
	return nil
}

func (hv *HexVal) String() string {
	return hex.EncodeToString(*hv)
}

func (hv *HexVal) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var content string
	if err := d.DecodeElement(&content, &start); err != nil {
		return err
	}
	return hv.FromString(content)
}

func (hv *HexVal) UnmarshalText(text []byte) (err error) {
	return hv.FromString(string(text))
}

// App is an Android application
type App struct {
	ID        string    `xml:"id"`
	Name      string    `xml:"name"`
	Summary   string    `xml:"summary"`
	Desc      string    `xml:"desc"`
	License   string    `xml:"license"`
	Categs    CommaList `xml:"categories"`
	Website   string    `xml:"web"`
	Source    string    `xml:"source"`
	Tracker   string    `xml:"tracker"`
	Changelog string    `xml:"changelog"`
	Donate    string    `xml:"donate"`
	Bitcoin   string    `xml:"bitcoin"`
	Litecoin  string    `xml:"litecoin"`
	Dogecoin  string    `xml:"dogecoin"`
	FlattrID  string    `xml:"flattr"`
	Apks      []Apk     `xml:"package"`
	CVName    string    `xml:"marketversion"`
	CVCode    int       `xml:"marketvercode"`
	CurApk    *Apk
}

type HexHash struct {
	Type string `xml:"type,attr"`
	Data HexVal `xml:",chardata"`
}

type DateVal struct {
	time.Time
}

func (dv *DateVal) FromString(s string) error {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}
	*dv = DateVal{t}
	return nil
}

func (dv *DateVal) String() string {
	return dv.Format("2006-01-02")
}

func (dv *DateVal) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var content string
	if err := d.DecodeElement(&content, &start); err != nil {
		return err
	}
	return dv.FromString(content)
}

func (dv *DateVal) UnmarshalText(text []byte) (err error) {
	return dv.FromString(string(text))
}

// Apk is an Android package
type Apk struct {
	VName   string    `xml:"version"`
	VCode   int       `xml:"versioncode"`
	Size    int64     `xml:"size"`
	MinSdk  int       `xml:"sdkver"`
	MaxSdk  int       `xml:"maxsdkver"`
	ABIs    CommaList `xml:"nativecode"`
	ApkName string    `xml:"apkname"`
	SrcName string    `xml:"srcname"`
	Sig     HexVal    `xml:"sig"`
	Added   DateVal   `xml:"added"`
	Perms   CommaList `xml:"permissions"`
	Feats   CommaList `xml:"features"`
	Hash    HexHash   `xml:"hash"`
}

func (app *App) TextDesc(w io.Writer) {
	reader := strings.NewReader(app.Desc)
	decoder := xml.NewDecoder(reader)
	firstParagraph := true
	linePrefix := ""
	colsUsed := 0
	var links []string
	linked := false
	for {
		token, err := decoder.Token()
		if err == io.EOF || token == nil {
			break
		}
		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				if firstParagraph {
					firstParagraph = false
				} else {
					fmt.Fprintln(w)
				}
				linePrefix = ""
				colsUsed = 0
			case "li":
				fmt.Fprint(w, "\n *")
				linePrefix = "   "
				colsUsed = 0
			case "a":
				for _, attr := range t.Attr {
					if attr.Name.Local == "href" {
						links = append(links, attr.Value)
						linked = true
						break
					}
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "p":
				fmt.Fprintln(w)
			case "ul":
				fmt.Fprintln(w)
			case "ol":
				fmt.Fprintln(w)
			}
		case xml.CharData:
			left := string(t)
			if linked {
				left += fmt.Sprintf("[%d]", len(links)-1)
				linked = false
			}
			limit := 80 - len(linePrefix) - colsUsed
			firstLine := true
			for len(left) > limit {
				last := 0
				for i, c := range left {
					if i >= limit {
						break
					}
					if c == ' ' {
						last = i
					}
				}
				if firstLine {
					firstLine = false
					limit += colsUsed
				} else {
					fmt.Fprint(w, linePrefix)
				}
				fmt.Fprintln(w, left[:last])
				left = left[last+1:]
				colsUsed = 0
			}
			if firstLine {
				firstLine = false
			} else {
				fmt.Fprint(w, linePrefix)
			}
			fmt.Fprint(w, left)
			colsUsed += len(left)
		}
	}
	if len(links) > 0 {
		fmt.Fprintln(w)
		for i, link := range links {
			fmt.Fprintf(w, "[%d] %s\n", i, link)
		}
	}
}

type appList []App

func (al appList) Len() int           { return len(al) }
func (al appList) Swap(i, j int)      { al[i], al[j] = al[j], al[i] }
func (al appList) Less(i, j int) bool { return al[i].ID < al[j].ID }

type apkList []Apk

func (al apkList) Len() int           { return len(al) }
func (al apkList) Swap(i, j int)      { al[i], al[j] = al[j], al[i] }
func (al apkList) Less(i, j int) bool { return al[i].VCode > al[j].VCode }

func LoadIndexXml(r io.Reader) (*Index, error) {
	var index Index
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&index); err != nil {
		return nil, err
	}

	sort.Sort(appList(index.Apps))

	for i := range index.Apps {
		app := &index.Apps[i]
		sort.Sort(apkList(app.Apks))
		app.calcCurApk()
	}
	return &index, nil
}

func (app *App) calcCurApk() {
	for _, apk := range app.Apks {
		app.CurApk = &apk
		if app.CVCode >= apk.VCode {
			break
		}
	}
}
