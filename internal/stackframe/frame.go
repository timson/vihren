//
// Copyright (C) 2026 Tim Sleptsov
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package stackframe

import (
	"regexp"
	"strings"
)

var (
	suffixesToTruncate = regexp.MustCompile(
		"_\\[j\\]$|_\\[j\\]_\\[s\\]$|_\\[i\\]$|_\\[i\\]_\\[s\\]$|_\\[0\\]$|_\\[0\\]_\\[s\\]$|_\\[1\\]$|_\\[1\\]_\\[s" +
			"\\]$|_\\[p\\]$|_\\[pe\\]$|_\\[k\\]$|_\\[php\\]$|_\\[pn\\]$|_\\[rb\\]$|_\\[net\\]$")
)

type Frame struct {
	Hash         int64
	ParentHash   int64
	Name         string
	Suffix       string
	Samples      int64
	IsRoot       bool
	Children     map[int64]bool
	Lang         string
	SpecialType  string
	IsThirdParty string
	Insight      string
}

func (f *Frame) GetTruncatedNameAndSuffix() (string, string) {
	if f.Lang == Cpp || f.Lang == NodeJS || f.Lang == Go {
		return f.Name, ""
	}
	suffix := suffixesToTruncate.FindString(f.Name)
	truncatedName := strings.TrimSuffix(f.Name, suffix)
	return truncatedName, suffix
}
