//
// Copyright (C) 2021, Stanislaw Adaszewski
// See LICENSE for terms
//

package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"github.com/go-errors/errors"
)

type JailTypeRaw interface {
	GetNode() *node32
	WriteTo(io.Writer)
}

type JailEntryRaw struct {
	Node *node32
	Text string
}

type JailConfRaw struct {
	Entries []JailTypeRaw
}

type JailBlockRaw struct {
	Node *node32
	Entries []JailEntryRaw
}

func (entry JailEntryRaw) GetNode() *node32 {
	return entry.Node
}

func (jblk JailBlockRaw) GetNode() *node32 {
	return jblk.Node
}

func (conf JailConfRaw) WriteTo(w io.Writer) {
	for _, e := range conf.Entries {
		e.WriteTo(w)
	}
}

func (entry JailEntryRaw) WriteTo(w io.Writer) {
	io.WriteString(w, entry.Text)
}

func (parser *JailConfParser) GetJailBlock(conf JailConfRaw, name string) (JailBlock, error) {
	for _, e := range conf.Entries {
		switch e.(type) {
		case JailBlockRaw:
			res := parser.ToBlock(e.(JailBlockRaw))
			if res.Name == name {
				return res, nil
			}
		}
	}
	return JailBlock{}, errors.New("Jail not found")
}

func (parser *JailConfParser) RemoveJailBlock(conf JailConfRaw, name string) (JailConfRaw, bool) {
	res := JailConfRaw{}
	wasRemoved := false
	for _, e := range conf.Entries {
		switch e.(type) {
		case JailBlockRaw:
			jblk := parser.ToBlock(e.(JailBlockRaw))
			if jblk.Name == name {
				wasRemoved = true
				continue
			} else {
				res.Entries = append(res.Entries, e)
			}
		default:
			res.Entries = append(res.Entries, e)
		}
	}
	return res, wasRemoved
}

func (jblk JailBlockRaw) WriteTo(w io.Writer) {
	for _, e := range jblk.Entries {
		e.WriteTo(w)
	}
}

func (parser *JailConfParser) ToBlock(jblk JailBlockRaw) JailBlock {
	res := parser.HandleJailBlock(jblk.Node)
	return res
}

func (parser *JailConfParser) ToRawConf() JailConfRaw {
	return parser.HandleTopRaw(parser.AST())
}

func (parser *JailConfParser) HandleTopRaw(node *node32) JailConfRaw {
	if (node.pegRule != ruletop) {
		panic("Expected top")
	}
	node = node.up
	res := JailConfRaw{}
	for node != nil {
		switch node.pegRule {
		case rulejail_block:
			res.Entries = append(res.Entries,
				parser.HandleJailBlockRaw(node))
		default:
			res.Entries = append(res.Entries,
				JailEntryRaw{Node: node,
					Text: parser.Buffer[node.begin:node.end]})
		}
		node = node.next
	}
	return res
}

func (parser *JailConfParser) HandleJailBlockRaw(node *node32) JailBlockRaw {
	if (node.pegRule != rulejail_block) {
		panic("Expected jail block")
	}
	res := JailBlockRaw{ Node: node }
	node = node.up
	for node != nil {
		res.Entries = append(res.Entries,
			JailEntryRaw{Node: node,
				Text: parser.Buffer[node.begin:node.end]})
		node = node.next
	}
	return res
}

func (jblk JailBlock) ToRaw() JailBlockRaw {
	res := JailBlockRaw{}
	res.Entries = append(res.Entries, JailEntryRaw{ Text: "\n" + jblk.Name + " {\n" })
	b := &strings.Builder{}
	for k, v := range jblk.KeyValuePairs {
		b.Reset()
		b.WriteString("  ")
		JailKeyValuePair{ Key: k, Value: v }.WriteTo(b)
		res.Entries = append(res.Entries, JailEntryRaw{ Text: b.String() })
	}
	res.Entries = append(res.Entries, JailEntryRaw{ Text: "}\n" })
	return res
}

func main() {
	expr, err := ioutil.ReadFile("samplejail.conf");
	if err != nil {
		log.Fatal(err)
	}
	// log.Println(string(expr))
	parser := &JailConfParser{Buffer: string(expr)}
	parser.Init()
	if err := parser.Parse(); err != nil {
		log.Fatal(err)
	}
	conf := parser.ToRawConf()
	// conf.WriteTo(os.Stdout)
	var jblk JailBlock
	if jblk, err = parser.GetJailBlock(conf, "foo"); err != nil {
		log.Fatal(err)
	}
	jblk.WriteTo(os.Stdout)

	//conf, _ = parser.RemoveJailBlock(conf, "foo")

	newJail := JailBlock{ Name: "lorem", KeyValuePairs: map[string]JailValue{
		"foo": JailValueFromString("bar"),
		"allow.mount": JailValueFromString("true"),
	} }
	conf.Entries = append(conf.Entries, newJail.ToRaw())

	//conf.WriteTo(os.Stdout)
}
