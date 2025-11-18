/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2025. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package trietree is for Prefix Matching
package trietree

import (
	"strings"
	"sync"

	"frontend/pkg/common/faas_common/constant"
)

// Trie -
type Trie struct {
	root *trieNode
	sync.RWMutex
}

type trieNode struct {
	children map[string]*trieNode
	isEnd    bool
}

// NewTrie -
func NewTrie() *Trie {
	return &Trie{
		root: &trieNode{
			children: make(map[string]*trieNode),
		},
	}
}

// Insert -
func (t *Trie) Insert(word []string) {
	t.Lock()
	defer t.Unlock()
	node := t.root
	for _, char := range word {
		if _, ok := node.children[char]; !ok {
			node.children[char] = &trieNode{
				children: make(map[string]*trieNode),
			}
		}
		node = node.children[char]
	}
	node.isEnd = true
}

// Search -
func (t *Trie) Search(word []string) bool {
	t.RLock()
	defer t.RUnlock()
	node := t.root
	for _, char := range word {
		if _, ok := node.children[char]; !ok {
			return false
		}
		node = node.children[char]
	}
	return node.isEnd
}

// Delete -
func (t *Trie) Delete(word []string) {
	t.Lock()
	t.delete(t.root, word, 0)
	t.Unlock()
}

func (t *Trie) delete(node *trieNode, word []string, depth int) bool {
	if depth == len(word) {
		if !node.isEnd {
			return false
		}
		node.isEnd = false
		// 如果删除后节点没有子节点了，可以删除这个节点
		return len(node.children) == 0
	}

	char := word[depth]
	childNode, ok := node.children[char]
	if !ok {
		return false
	}

	shouldDeleteChild := t.delete(childNode, word, depth+1)
	if shouldDeleteChild {
		delete(node.children, char)
		// 删除子节点后如果当前节点也没有其他子节点了，并且不是其他单词的结尾，可以删除当前节点
		return !node.isEnd && len(node.children) == 0
	}

	return false
}

// LongestMatch -
func (t *Trie) LongestMatch(s []string) string {
	t.RLock()
	defer t.RUnlock()
	node := t.root
	longestMatch := ""
	var currentMatch []string
	for _, char := range s {
		if child, ok := node.children[char]; ok {
			currentMatch = append(currentMatch, char)
			node = child
			if node.isEnd {
				longestMatch = strings.Join(currentMatch, constant.URLSeparator)
			}
		} else {
			break
		}
	}
	return longestMatch
}
