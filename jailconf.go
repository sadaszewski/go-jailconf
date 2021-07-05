//
// Copyright (C) 2021, Stanislaw Adaszewski
// See LICENSE for terms
//

package main

import (
	"log"
	"fmt"
	"strconv"
	"strings"
	"io"
	"os"
	"io/ioutil"
	// "github.com/go-errors/errors"
)

type JailValue struct {
	Items []string
}

type JailBlock struct {
	Name string
	KeyValuePairs map[string]JailValue
}

type JailKeyValuePair struct {
	Key string
	Value JailValue
}

type JailKeyValueAppendPair struct {
	Key string
	Value JailValue
}

type JailKeySet struct {
	Key string
}

type JailType interface {
	Print()
	WriteTo(io.Writer)
}

type CommentBlock struct {
	Comment string
}

type JailConf struct {
	Entries []JailType
}

func (ks JailKeySet) Print() {
	log.Printf("%s;\n", ks.Key)
}

func (ks JailKeySet) WriteTo(w io.Writer) {
	io.WriteString(w, ks.Key)
	io.WriteString(w, ";\n")
}

func (cb CommentBlock) Print() {
	log.Println(cb.Comment)
}

func (cb CommentBlock) WriteTo(w io.Writer) {
	io.WriteString(w, cb.Comment)
	io.WriteString(w, "\n")
}

func (conf JailConf) WriteTo(w io.Writer) {
  for _, e := range conf.Entries {
	  e.WriteTo(w)
  }
}

func (kvp JailKeyValuePair) WriteTo(w io.Writer) {
	io.WriteString(w, kvp.Key)
	io.WriteString(w, " = ")
	kvp.Value.WriteTo(w)
	io.WriteString(w, ";\n")
}

func (kvp JailKeyValueAppendPair) WriteTo(w io.Writer) {
	io.WriteString(w, kvp.Key)
	io.WriteString(w, " += ")
	kvp.Value.WriteTo(w)
	io.WriteString(w, ";\n")
}

func (jblk JailBlock) WriteTo(w io.Writer) {
	io.WriteString(w, EscapeString(jblk.Name))
	io.WriteString(w, " {\n")
	for k, v := range jblk.KeyValuePairs {
		io.WriteString(w, "  ")
		JailKeyValuePair{Key: k, Value: v}.WriteTo(w)
	}
	io.WriteString(w, "}\n")
}

func (value JailValue) WriteTo(w io.Writer) {
	io.WriteString(w, value.Sprint())
}

func EscapeString(s string) string {
	res := strconv.Quote(s)
	if ! strings.Contains(s, " ") && res[1:len(res)-1] == s {
		return s;
	}
	return res;
}

func (value JailValue) Sprint() string {
	escaped := make([]string, len(value.Items))
	for i, v := range value.Items {
		escaped[i] = EscapeString(v)
	}
	return strings.Join(escaped, ", ")
}

func (conf JailConf) Print() {
	for _, e := range conf.Entries {
		e.Print()
	}
}

func (kvp JailKeyValuePair) Print() {
	log.Println(fmt.Sprintf("JailKeyValuePair { Key: %s Value: %s }", kvp.Key, kvp.Value.Sprint() ))
}

func (kvp JailKeyValueAppendPair) Print() {
	log.Println(fmt.Sprintf("JailKeyValueAppendPair { Key: %s Value: %s }", kvp.Key, kvp.Value.Sprint() ))
}

func (jblock JailBlock) Print() {
	log.Println("JailBlock {")
	log.Printf("  Name: %s\n", jblock.Name)
	for k, v := range jblock.KeyValuePairs {
		log.Println(fmt.Sprintf("  Key: %s, Value: %s", k, v.Sprint()))
	}
	log.Println("}")
}

func NewJailBlock() JailBlock {
	res := JailBlock{}
	res.KeyValuePairs = make(map[string]JailValue)
	return res
}

func (value JailValue) Extend(other JailValue) JailValue {
	res := value
	res.Items = append(value.Items, other.Items...)
	return res
}

func (value JailValue) Item() string {
	if len(value.Items) != 1 {
		panic("Expected single item")
	}
	return value.Items[0]
}

func (parser *JailConfParser) ToStruct() JailConf {
	return parser.HandleTopRule(parser.AST())
}

func (parser *JailConfParser) HandleTopRule(node *node32) JailConf {
	if (node.pegRule != ruletop) {
		panic("Expected top expression")
	}
	jailConf := JailConf{}
	node = node.up
	for node != nil {
		// log.Println(rul3s[node.pegRule])
		switch(node.pegRule) {
		case rulejail_block:
			jailConf.Entries = append(jailConf.Entries,
				parser.HandleJailBlock(node))
		case rulekey_value_pair:
			jailConf.Entries = append(jailConf.Entries,
				parser.HandleKeyValuePair(node))
		case rulekey_value_append_pair:
			jailConf.Entries = append(jailConf.Entries,
				parser.HandleKeyValueAppendPair(node))
		case rulekey_set:
			jailConf.Entries = append(jailConf.Entries,
				parser.HandleKeySet(node))
		}
		node = node.next
	}
	return jailConf
}

func (parser *JailConfParser) HandleUnquotedString(node *node32) JailValue {
	if node.pegRule != ruleunquoted_string {
		panic("Expected unquoted string")
	}
	node = node.up
	text := ""
	for node != nil {
		switch(node.pegRule) {
		case ruleunquoted_safe_char:
			text += parser.Buffer[node.begin:node.end]
		}
		node = node.next
	}
	text, _ = strconv.Unquote("\"" + text + "\"")
	return JailValue{Items: []string{ text }}
}

func (parser *JailConfParser) HandleSingleQuotedString(node *node32) JailValue {
	if node.pegRule != rulesingle_quoted_string {
		panic("Expected single quoted string")
	}
	node = node.up
	text := ""
	for node != nil {
		switch node.pegRule {
			case ruleesc_sing_quote:
				text += "\\'"
			case ruleesc_backslash:
				text += "\\\\"
			case rulesing_quote_safe_char:
				text += parser.Buffer[node.begin:node.end]
		}
		node = node.next
	}
	text, _ = strconv.Unquote("\"" + text + "\"")
	return JailValue{Items: []string{ text }}
}

func (parser *JailConfParser) HandleDoubleQuotedString(node *node32) JailValue {
	if node.pegRule != ruledouble_quoted_string {
		panic("Expected double quoted string")
	}
	node = node.up
	text := ""
	for node != nil {
		switch node.pegRule {
			case ruleesc_dbl_quote:
				text += "\\\""
			case ruleesc_backslash:
				text += "\\\\"
			case ruledbl_quote_safe_char:
				text += parser.Buffer[node.begin:node.end]
		}
		node = node.next
	}
	text, _ = strconv.Unquote("\"" + text + "\"")
	return JailValue{Items: []string{ text }}
}

func (parser *JailConfParser) HandleString(node *node32) JailValue {
	node = node.up
	switch (node.pegRule) {
	case ruleunquoted_string:
		return parser.HandleUnquotedString(node)
	case rulequoted_string:
		node = node.up
		switch(node.pegRule) {
		case rulesingle_quoted_string:
			return parser.HandleSingleQuotedString(node)
		case ruledouble_quoted_string:
			return parser.HandleDoubleQuotedString(node)
		}
	}
	panic(fmt.Sprintf("Expected one of the string types, got: %s", rul3s[node.pegRule]))
}

func (parser *JailConfParser) HandleSingleValue(node *node32) JailValue {
	node = node.up
	if node.pegRule != rulestring {
		panic("Expected string")
	}
	return parser.HandleString(node)
}

func (parser *JailConfParser) HandleListOfValues(node *node32) JailValue {
	node = node.up
	res := JailValue{}
	for node != nil {
		if node.pegRule == rulesingle_value {
			res.Items = append(res.Items, parser.HandleSingleValue(node).Item())
		}
		node = node.next
	}
	return res
}

func (parser *JailConfParser) HandleValue(node *node32) JailValue {
	node = node.up
	switch node.pegRule {
	case rulesingle_value:
		return parser.HandleSingleValue(node)
	case rulelist_of_values:
		return parser.HandleListOfValues(node)
	}
	panic(fmt.Sprintf("Not supposed to happen: %s", rul3s[node.pegRule]))
}

func (parser *JailConfParser) HandleJailBlock(node *node32) JailBlock {
	node = node.up
	var jailName JailValue
	_ = jailName
	jailBlock := NewJailBlock()
	for node != nil {
		log.Println(rul3s[node.pegRule])
		switch(node.pegRule) {
		case rulejail_name:
			jailName = parser.HandleString(node.up)
		case rulekey_value_pair:
			kvp := parser.HandleKeyValuePair(node)
			jailBlock.KeyValuePairs[kvp.Key] = kvp.Value
		case rulekey_value_append_pair:
			kvp := parser.HandleKeyValueAppendPair(node)
			jailBlock.KeyValuePairs[kvp.Key] = jailBlock.KeyValuePairs[kvp.Key].Extend(kvp.Value)
		case rulekey_set:
			kvp := parser.HandleKeySet(node)
			jailBlock.KeyValuePairs[kvp.Key] = kvp.Value
		}
		node = node.next
	}
	jailBlock.Name = jailName.Item()
	return jailBlock
}

func (parser *JailConfParser) HandleKeyValuePair(node *node32) JailKeyValuePair {
	if node.pegRule != rulekey_value_pair {
		panic("Expected key value pair")
	}
	node = node.up
	var key string
	var value JailValue
	_, _ = key, value
	for node != nil {
		switch(node.pegRule) {
		case rulekey:
			log.Printf("key node: %s", rul3s[node.up.pegRule])
			key = parser.HandleString(node.up).Item()
		case rulevalue:
			value = parser.HandleValue(node)
		}
		node = node.next
	}
	return JailKeyValuePair{Key: key, Value: value}
}

func (parser *JailConfParser) HandleKeyValueAppendPair(node *node32) JailKeyValueAppendPair {
	if node.pegRule != rulekey_value_append_pair {
		panic("Expected key value append pair")
	}
	node = node.up
	var key string
	var value JailValue
	_, _ = key, value
	for node != nil {
		switch(node.pegRule) {
		case rulekey:
			key = parser.HandleString(node.up).Item()
		case rulevalue:
			value = parser.HandleValue(node)
		}
		node = node.next
	}
	return JailKeyValueAppendPair{Key: key, Value: value}
}

func GetKeySetValue(key string) (string, string) {
	res := strings.Split(key, ".")
	isNegative := strings.HasPrefix(res[len(res)-1], "no")
	if isNegative {
		res[len(res)-1] = res[len(res)-1][2:]
		return strings.Join(res, "."), "false"
	} else {
		return key, "true"
	}
}

func (parser *JailConfParser) HandleKeySet(node *node32) JailKeyValuePair {
	if node.pegRule != rulekey_set {
		panic("Expected key set statement")
	}
	node = node.up
	var key string
	var value string
	_, _ = key, value
	for node != nil {
		switch(node.pegRule) {
		case rulekey:
			key = parser.HandleString(node.up).Item()
		}
		node = node.next
	}
	key, value = GetKeySetValue(key)
	return JailKeyValuePair{Key: key, Value: JailValueFromString(value)}
}

func JailValueFromString(s string) JailValue {
	return JailValue{Items: []string { s }}
}

func main2() {
	expression := "/* a little comment \nto begin with */ foo = bar; baf=\"line\\\ncontinues\"; // end of line comment\n # shell style comment\n'my jail name with space' { bar.baf.baz = 1; baz.bee.boo = 1,2,3; baz.bee.boo += 4; broken.param = first_line\\\nsecond_line_embedded_newline\\n_here; }"
	log.Println(expression)
	parser1 := JailConfParser{Buffer: expression}
	parser := &parser1;
	(&parser1).Init()
	if err := parser.Parse(); err != nil {
		log.Fatal(err)
	}
	// parser.PrintSyntaxTree()
	conf := parser.ToStruct()
	conf.Print()
	conf.Entries = append(conf.Entries, JailKeyValuePair{Key: "appended_key", Value: JailValueFromString("appended_value")})
	conf.WriteTo(os.Stdout)

	expr, err := ioutil.ReadFile("samplejail.conf");
	if err != nil {
		log.Fatal(err)
	}
	log.Println(string(expr))
	parser = &JailConfParser{Buffer: string(expr)}
	parser.Init()
	if err := parser.Parse(); err != nil {
		log.Fatal(err)
	}
	conf = parser.ToStruct()
	conf.WriteTo(os.Stdout)
}
