//
// Copyright (C) 2023 Intel Corporation
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

package indexer

import (
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type FrameNormalizer struct {
	rules         []*regexp.Regexp
	replacements  []string
	anyRuleRegexp *regexp.Regexp
}

func NewFrameNormalizer(rulesPath string) (*FrameNormalizer, error) {
	n := &FrameNormalizer{}
	if err := n.LoadFromFile(rulesPath); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *FrameNormalizer) LoadFromFile(rulesPath string) error {
	cfg, err := LoadRules(rulesPath)
	if err != nil {
		return err
	}

	ruleCount := len(cfg.Rules)
	rules := make([]*regexp.Regexp, 0, ruleCount)
	replacements := make([]string, 0, ruleCount)
	patterns := make([]string, 0, ruleCount)

	for _, rule := range cfg.Rules {
		if len(rule.Tests) == 0 {
			return errors.Errorf("rule has no tests: %+v", rule)
		}

		re, err := regexp.Compile(rule.Regexp)
		if err != nil {
			return err
		}

		for _, tc := range rule.Tests {
			matched := re.MatchString(tc.Input)
			if matched {
				got := re.ReplaceAllLiteralString(tc.Input, rule.Replace)
				if got != tc.Output {
					return errors.Errorf("rule test failed (pattern=%q): expected %q, got %q", rule.Regexp, tc.Output, got)
				}
			} else if !tc.ShouldNotMatch {
				return errors.Errorf("rule test expected match but did not (pattern=%q, input=%q)", rule.Regexp, tc.Input)
			}
		}

		rules = append(rules, re)
		replacements = append(replacements, rule.Replace)
		patterns = append(patterns, rule.Regexp)
	}

	any, err := regexp.Compile(strings.Join(patterns, "|"))
	if err != nil {
		return err
	}

	n.rules = rules
	n.replacements = replacements
	n.anyRuleRegexp = any

	log.WithFields(log.Fields{
		"path":  rulesPath,
		"rules": len(rules),
	}).Info("frame normalizer loaded")
	return nil
}

func (n *FrameNormalizer) ShouldNormalize(s string) bool {
	return n.anyRuleRegexp != nil && n.anyRuleRegexp.MatchString(s)
}

func (n *FrameNormalizer) Normalize(s string) string {
	for i, re := range n.rules {
		if re.MatchString(s) {
			s = re.ReplaceAllLiteralString(s, n.replacements[i])
		}
	}
	return s
}

func LoadRules(path string) (*Rules, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Rules
	if err = yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}

	for i := range cfg.Rules {
		re, err := regexp.Compile(cfg.Rules[i].Regexp)
		if err != nil {
			return &cfg, err
		}
		cfg.Rules[i].CompiledRegexp = re
	}

	return &cfg, nil
}
