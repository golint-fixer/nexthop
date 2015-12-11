package command

import (
	//"bufio"
	"fmt"
	"log"
	"strings"
	//"text/scanner"
)

const (
	MOTD = iota
	USER = iota
	PASS = iota
	EXEC = iota
	ENAB = iota
	CONF = iota
)

type CmdClient interface {
	ConfigPath() string
	ConfigPathSet(path string)
	Send(msg string)
	Sendln(msg string)
	SendlnNow(msg string)
	InputQuit()
	Output() chan<- string
}

type CmdFunc func(ctx ConfContext, node *CmdNode, line string, c CmdClient)

const (
	CMD_NONE = uint64(0)
	CMD_CONF = uint64(1 << 0)
)

type CmdNode struct {
	Path     string
	Desc     string
	MinLevel int
	Handler  CmdFunc
	Children []*CmdNode
	Options  uint64
}

func (n *CmdNode) IsConfig() bool {
	return n.Options&CMD_CONF != 0
}

type ConfNode struct {
	Path     string
	Value    []string
	Children []*ConfNode
}

func (n *ConfNode) ValueAdd(value string) error {
	for _, v := range n.Value {
		if v == value {
			return nil // already exists
		}
	}
	n.Value = append(n.Value, value) // append new
	return nil
}

func (n *ConfNode) ValueSet(value string) {
	n.Value = []string{value}
}

func (n *ConfNode) Set(path, line string) (*ConfNode, error, bool) {

	expanded, err := cmdExpand(line, path)
	if err != nil {
		return nil, fmt.Errorf("ConfNode.Set error: %v", err), false
	}

	log.Printf("ConfNode.Set: line=[%v] path=[%v] expand=[%s]", line, path, expanded)

	labels := strings.Fields(expanded)
	size := len(labels)
	parent := n
	for i, label := range labels {
		child := parent.findChild(label)
		if child != nil {
			// found, search next
			parent = child
			continue
		}

		// not found

		for ; i < size-1; i++ {
			// intermmediate label
			label = labels[i]
			currPath := strings.Join(labels[:i+1], " ")
			newNode := &ConfNode{Path: currPath}
			parent.Children = append(parent.Children, newNode)
			parent = newNode
		}

		// last label
		label = labels[size-1]
		newNode := &ConfNode{Path: expanded}
		parent.Children = append(parent.Children, newNode)

		return newNode, nil, false
	}

	// existing node found

	return parent, nil, true
}

func (n *ConfNode) Get(path string) (*ConfNode, error) {

	labels := strings.Fields(path)
	parent := n
	for _, label := range labels {
		child := parent.findChild(label)
		if child != nil {
			// found, search next
			parent = child
			continue
		}

		// not found
		return nil, fmt.Errorf("ConfNode.Get not found")
	}

	return parent, nil // found
}

func (n *ConfNode) findChild(label string) *ConfNode {

	for _, c := range n.Children {
		last := LastToken(c.Path)
		if label == last {
			return c
		}
	}

	return nil
}

type ConfContext interface {
	CmdRoot() *CmdNode
	ConfRootCandidate() *ConfNode
	ConfRootActive() *ConfNode

	Hostname() string
}

/*
func firstToken(path string) string {
	// fixme with tokenizer
	return strings.Fields(path)[0]
}
*/

func LastToken(path string) string {
	// fixme with tokenizer
	f := strings.Fields(path)
	return f[len(f)-1]
}

func StripLastToken(path string) (string, string) {
	last := strings.LastIndexByte(path, ' ')
	return path[:last], path[last+1:]
}

func dumpChildren(node *CmdNode) string {
	str := ""
	for _, p := range node.Children {
		str += fmt.Sprintf(",%s", p.Path)
	}
	return str
}

func pushChild(node, child *CmdNode) {
	//log.Printf("pushChild: parent=[%s] child=[%s] before: [%v]", node.Path, child.Path, dumpChildren(node))
	node.Children = append(node.Children, child)
	//log.Printf("pushChild: parent=[%s] child=[%s] after: [%v]", node.Path, child.Path, dumpChildren(node))
}

func CmdInstall(root *CmdNode, opt uint64, path string, min int, cmd CmdFunc, desc string) {
	if _, err := cmdAdd(root, opt, path, min, cmd, desc); err != nil {
		log.Printf("cmdInstall: error %s", err)
	}
}

func cmdAdd(root *CmdNode, opt uint64, path string, min int, cmd CmdFunc, desc string) (*CmdNode, error) {
	//log.Printf("cmdInstall: [%s]", path)

	labelList := strings.Fields(path)
	size := len(labelList)
	parent := root
	for i, label := range labelList {
		currPath := strings.Join(labelList[:i+1], " ")
		//log.Printf("cmdInstall: %d: curr=[%s] label=[%s]", i, currPath, label)
		child := findChild(parent, label)
		if child != nil {
			// found, search next
			//log.Printf("cmdInstall: found [%s]", currPath)
			parent = child
			continue
		}

		// not found

		//log.Printf("cmdInstall: new [%s]", currPath)

		for ; i < size-1; i++ {
			// intermmediate label
			label = labelList[i]
			currPath = strings.Join(labelList[:i+1], " ")
			//log.Printf("cmdInstall: %d: intermmediate curr=[%s] label=[%s]", i, currPath, label)
			newNode := &CmdNode{Path: currPath, MinLevel: min, Options: opt}
			pushChild(parent, newNode)
			parent = newNode
		}

		// last label
		label = labelList[size-1]
		//log.Printf("cmdInstall: %d: leaf curr=[%s] label=[%s]", i, path, label)
		newNode := &CmdNode{Path: path, Desc: desc, MinLevel: min, Handler: cmd, Options: opt}
		pushChild(parent, newNode)

		// did this command create an unreachable location?

		n, err := CmdFind(root, path, CONF)
		if err != nil {
			return newNode, fmt.Errorf("root=[%s] cmd=[%s] created unreachable command node: %v", root.Path, path, err)
		}

		if n != newNode {
			return newNode, fmt.Errorf("root=[%s] cmd=[%s] created wrong command node: %v", root.Path, path, err)
		}

		return newNode, nil
	}

	// command node found

	return parent, fmt.Errorf("[%s] already exists", path)
}

func findChild(node *CmdNode, label string) *CmdNode {

	for _, c := range node.Children {
		last := LastToken(c.Path)
		//log.Printf("findChild: searching [%s] against (%s)[%s] under [%s]", label, last, c.Path, node.Path)
		if label == last {
			//log.Printf("findChild: found [%s] as [%s] under [%s]", label, c.Path, node.Path)
			return c
		}
	}

	//log.Printf("findChild: not found [%s] under [%s]", label, node.Path)

	return nil
}

func isConfigValueKeyword(str string) bool {
	return str[0] == '{' && str[len(str)-1] == '}'
}

func matchChildren(children []*CmdNode, label string) []*CmdNode {
	c := []*CmdNode{}

	for _, n := range children {
		last := LastToken(n.Path)
		if isConfigValueKeyword(last) {
			// these keywords match any label
			c = append(c, n)
			continue
		}
		if strings.HasPrefix(last, label) {
			c = append(c, n)
			continue
		}
	}

	return c
}

func CmdFindRelative(root *CmdNode, line, configPath string, status int) (*CmdNode, string, error) {

	prependConfigPath := true // assume it's a config cmd
	n, e := CmdFind(root, line, status)
	if e == nil {
		// found at root
		if !n.IsConfig() {
			// not a config cmd -- ignore prepend path
			prependConfigPath = false
		}
	}

	var lookupPath string
	if prependConfigPath && configPath != "" {
		// prepend path to config command
		lookupPath = fmt.Sprintf("%s %s", configPath, line)
	} else {
		lookupPath = line
	}

	node, err := CmdFind(root, lookupPath, status)
	if err != nil {
		return nil, lookupPath, fmt.Errorf("dispatchCommand: command not found: %s", err)
	}

	if node.MinLevel > status {
		return nil, lookupPath, fmt.Errorf("dispatchCommand: command level prohibited: [%s]", lookupPath)
	}

	return node, lookupPath, nil
}

func CmdFind(root *CmdNode, path string, level int) (*CmdNode, error) {

	/*
		var s scanner.Scanner
		s.Error = func(s *scanner.Scanner, msg string) {
			log.Printf("command scan error: %s [%s]", msg, path)
		}
		s.Init(strings.NewReader(path))
	*/

	tokens := strings.Fields(path)

	parent := root
	//for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
	for _, label := range tokens {
		//log.Printf("cmdFind: token: [%s]", s.TokenText())
		//label := s.TokenText()
		//log.Printf("cmdFind: token: [%s]", label)

		if len(parent.Children) == 1 && LastToken(parent.Children[0].Path) == "{ANY}" {
			// {ANY} is special construct for consuming anything
			return parent.Children[0], nil // found
		}

		children := matchChildren(parent.Children, label)
		size := len(children)
		if size < 1 {
			return nil, fmt.Errorf("CmdFind: not found: [%s] under [%s]", label, parent.Path)
		}
		if size > 1 {
			return nil, fmt.Errorf("CmdFind: ambiguous: [%s] under [%s]", label, parent.Path)
		}
		parent = children[0]
	}

	return parent, nil // found
}

// originalLine:    int eth0 ipv4 addr 1.1.1.1/30
// commandFullPath: interface {IFNAME} ipv4 address {ADDR}
// output:          interface eth0 ipv4 address 1.1.1.1/30
func cmdExpand(originalLine, commandFullPath string) (string, error) {
	lineFields := strings.Fields(originalLine)
	pathFields := strings.Fields(commandFullPath)

	lineLen := len(lineFields)
	pathLen := len(pathFields)

	if len(lineFields) != len(pathFields) {
		return "", fmt.Errorf("cmdExpand: length mismatch: line=%d path=%d", lineLen, pathLen)
	}

	for i, label := range pathFields {
		if label[0] == '{' {
			pathFields[i] = lineFields[i]
			continue
		}
	}

	return strings.Join(pathFields, " "), nil
}

func List(node *CmdNode, depth int, c CmdClient) {
	handler := "----"
	if node.Handler != nil {
		handler = "LEAF"
	}
	ident := strings.Repeat(" ", 4*depth)
	output := fmt.Sprintf("%s %d %s[%s] desc=[%s]", handler, node.MinLevel, ident, node.Path, node.Desc)
	log.Printf(output)
	c.Sendln(output)
	for _, n := range node.Children {
		List(n, depth+1, c)
	}
}
