package tools

import (
	"container/list"
	"fmt"
	"time"
)

// ListElement represent a record in list buffer
type ListElement struct {
	k interface{}
	v interface{}
	e *list.Element
	l *ListBuffer
	t time.Time // element creation time
}

// create a new ListElement instance
func newListElement(k, v interface{}, e *list.Element, l *ListBuffer) *ListElement {
	return &ListElement{
		k: k,
		v: v,
		e: e,
		l: l,
		t: time.Now(),
	}
}

// Get the element's key
func (self *ListElement) Key() interface{} {
	return self.k
}

// Get the element's value
func (self *ListElement) Value() interface{} {
	return self.v
}

// Get the next element
func (self *ListElement) Next() *ListElement {
	e := self.e.Next()
	if e == nil {
		return nil
	}
	return self.l.elements[e.Value]
}

// Get the element creation time
func (self *ListElement) CreationTime() time.Time {
	return self.t
}

// ListBuffer is a list buffer implementation.
type ListBuffer struct {
	elements map[interface{}]*ListElement
	keyList  *list.List
}

// NewListBuffer create a list buffer instance
func NewListBuffer() *ListBuffer {
	return &ListBuffer{
		elements: make(map[interface{}]*ListElement),
		keyList:  list.New(),
	}
}

// AddElement add an element to list buffer
func (self *ListBuffer) AddElement(key interface{}, value interface{}) error {
	if self.elements[key] != nil {
		return fmt.Errorf("record with key %v already in buffer", key)
	}
	e := self.keyList.PushBack(key)
	le := newListElement(key, value, e, self)
	self.elements[key] = le
	return nil
}

// GetElement get an element from list buffer
func (self *ListBuffer) GetElement(key interface{}) *ListElement {
	return self.elements[key]
}

// RemoveElement remove an element from list buffer
func (self *ListBuffer) RemoveElement(elem *ListElement) {
	k := elem.Key()
	if self.elements[k] != nil {
		delete(self.elements, k)
		self.keyList.Remove(elem.e)
	}
}

// RemoveElement remove an element from list buffer
func (self *ListBuffer) RemoveElementByKey(key interface{}) {
	if elem := self.elements[key]; elem != nil {
		delete(self.elements, key)
		self.keyList.Remove(elem.e)
	}
}

// Front returns the first element of ListBuffer l or nil if the list is empty.
func (self *ListBuffer) Front() *ListElement {
	if fk := self.keyList.Front(); fk != nil {
		return self.elements[fk.Value]
	}
	return nil
}

// Len returns the number of elements of ListBuffer.
func (self *ListBuffer) Len() int {
	return self.keyList.Len()
}
