package main

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	ErrQueueBlocked = errors.New("queue is blocked")
)

type Queue struct {
	items []interface{}
	mu    sync.Mutex
	block bool
}

func NewQueue() *Queue {
	return &Queue{
		items: make([]interface{}, 0),
		block: false,
	}
}

func (q *Queue) Append(item interface{}) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.block {
		return ErrQueueBlocked
	}

	q.items = append(q.items, item)
	return nil
}

func (q *Queue) GetCommaSepratedValues() string {
	strArr := make([]string, len(q.items))
	for i, v := range q.items {
		strArr[i] = fmt.Sprintf("%v", v)
	}
	return strings.Join(strArr, ",")
}

func (q *Queue) SetBlock(block bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.block = block
}

func (q *Queue) IsBlocked() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.block
}
