//Copyright 2015 Red Hat Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package doc

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anaminus/cobra"
)

func printOptionsReST(buf *bytes.Buffer, cmd *cobra.Command, name string) error {
	flags := cmd.NonInheritedFlags()
	flags.SetOutput(buf)
	if flags.HasAvailableFlags() {
		buf.WriteString("Options\n")
		buf.WriteString("~~~~~~~\n\n::\n\n")
		flags.PrintDefaults()
		buf.WriteString("\n")
	}

	parentFlags := cmd.InheritedFlags()
	parentFlags.SetOutput(buf)
	if parentFlags.HasAvailableFlags() {
		buf.WriteString("Options inherited from parent commands\n")
		buf.WriteString("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n\n::\n\n")
		parentFlags.PrintDefaults()
		buf.WriteString("\n")
	}
	return nil
}

// linkHandler for default ReST hyperlink markup
func defaultLinkHandler(name, ref string) string {
	return fmt.Sprintf("`%s <%s.rst>`_", name, ref)
}

// GenReST creates reStructured Text output.
func GenReST(cmd *cobra.Command, w io.Writer) error {
	return GenReSTCustom(cmd, w, defaultLinkHandler)
}

// GenReSTCustom creates custom reStructured Text output.
func GenReSTCustom(cmd *cobra.Command, w io.Writer, linkHandler func(string, string) string) error {
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()

	buf := new(bytes.Buffer)
	name := cmd.CommandPath()

	short := cmd.Short
	long := cmd.Long
	if len(long) == 0 {
		long = short
	}
	ref := strings.Replace(name, " ", "_", -1)

	buf.WriteString(".. _" + ref + ":\n\n")
	buf.WriteString(name + "\n")
	buf.WriteString(strings.Repeat("-", len(name)) + "\n\n")
	buf.WriteString(short + "\n\n")
	buf.WriteString("Synopsis\n")
	buf.WriteString("~~~~~~~~\n\n")
	buf.WriteString("\n" + long + "\n\n")

	if cmd.Runnable() {
		buf.WriteString(fmt.Sprintf("::\n\n  %s\n\n", cmd.UseLine()))
	}

	if len(cmd.Example) > 0 {
		buf.WriteString("Examples\n")
		buf.WriteString("~~~~~~~~\n\n")
		buf.WriteString(fmt.Sprintf("::\n\n%s\n\n", indentString(cmd.Example, "  ")))
	}

	if err := printOptionsReST(buf, cmd, name); err != nil {
		return err
	}
	if hasSeeAlso(cmd) {
		buf.WriteString("SEE ALSO\n")
		buf.WriteString("~~~~~~~~\n\n")
		if cmd.HasParent() {
			parent := cmd.Parent()
			pname := parent.CommandPath()
			ref = strings.Replace(pname, " ", "_", -1)
			buf.WriteString(fmt.Sprintf("* %s \t - %s\n", linkHandler(pname, ref), parent.Short))
		}

		children := cmd.Commands()
		sort.Sort(byName(children))

		for _, child := range children {
			if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
				continue
			}
			cname := name + " " + child.Name()
			ref = strings.Replace(cname, " ", "_", -1)
			buf.WriteString(fmt.Sprintf("* %s \t - %s\n", linkHandler(cname, ref), child.Short))
		}
		buf.WriteString("\n")
	}
	_, err := buf.WriteTo(w)
	return err
}

// GenReSTTree will generate a ReST page for this command and all
// descendants in the directory given.
// This function may not work correctly if your command names have `-` in them.
// If you have `cmd` with two subcmds, `sub` and `sub-third`,
// and `sub` has a subcommand called `third`, it is undefined which
// help output will be in the file `cmd-sub-third.1`.
func GenReSTTree(cmd *cobra.Command, dir string) error {
	emptyStr := func(s string) string { return "" }
	return GenReSTTreeCustom(cmd, dir, emptyStr, defaultLinkHandler)
}

// GenReSTTreeCustom is the the same as GenReSTTree, but
// with custom filePrepender and linkHandler.
func GenReSTTreeCustom(cmd *cobra.Command, dir string, filePrepender func(string) string, linkHandler func(string, string) string) error {
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if err := GenReSTTreeCustom(c, dir, filePrepender, linkHandler); err != nil {
			return err
		}
	}

	basename := strings.Replace(cmd.CommandPath(), " ", "_", -1) + ".rst"
	filename := filepath.Join(dir, basename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.WriteString(f, filePrepender(filename)); err != nil {
		return err
	}
	if err := GenReSTCustom(cmd, f, linkHandler); err != nil {
		return err
	}
	return nil
}

// adapted from: https://github.com/kr/text/blob/main/indent.go
func indentString(s, p string) string {
	var res []byte
	b := []byte(s)
	prefix := []byte(p)
	bol := true
	for _, c := range b {
		if bol && c != '\n' {
			res = append(res, prefix...)
		}
		res = append(res, c)
		bol = c == '\n'
	}
	return string(res)
}
