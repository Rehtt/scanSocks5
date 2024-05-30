package main

import "sync"

type Thread struct {
	c  chan struct{}
	wg sync.WaitGroup
}

func NewThread(limit int) *Thread {
	return &Thread{
		c: make(chan struct{}, limit),
	}
}

func (t *Thread) Run(f func(map[string]any), arg map[string]any) {
	t.c <- struct{}{}
	t.wg.Add(1)
	go func(t *Thread, f func(arg map[string]any), arg map[string]any) {
		f(arg)
		<-t.c
		t.wg.Done()
	}(t, f, arg)
}

func (t *Thread) Wait() {
	t.wg.Wait()
}
