package tools

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewListBuffer(t *testing.T) {
	assert := assert.New(t)
	lb := NewListBuffer()
	assert.NotNil(lb)
	assert.Equal(0, lb.Len())
}

func TestListBuffer_AddElement(t *testing.T) {
	assert := assert.New(t)
	lb := NewListBuffer()
	assert.NotNil(lb)
	assert.Nil(lb.AddElement("1", "1"))
	assert.NotNil(lb.AddElement("1", "1"))
}

func TestListBuffer_GetElement(t *testing.T) {
	assert := assert.New(t)
	lb := NewListBuffer()
	assert.NotNil(lb)
	assert.Nil(lb.AddElement("1", "1"))
	assert.NotNil(lb.GetElement("1"))
}

func TestListBuffer_RemoveElement(t *testing.T) {
	assert := assert.New(t)
	lb := NewListBuffer()
	assert.NotNil(lb)
	assert.Nil(lb.AddElement("1", "1"))
	e := lb.GetElement("1")
	assert.NotNil(e)
	lb.RemoveElement(e)
	assert.Nil(lb.GetElement("1"))
}

func TestListBuffer_Front(t *testing.T) {
	assert := assert.New(t)
	lb := NewListBuffer()
	assert.NotNil(lb)
	assert.Nil(lb.AddElement("1", "1"))
	assert.Nil(lb.AddElement("2", "2"))
	assert.Nil(lb.AddElement("3", "3"))
	e := lb.Front()
	assert.Equal("1", e.Key())
}

func TestListBuffer_Len(t *testing.T) {
	assert := assert.New(t)
	lb := NewListBuffer()
	assert.NotNil(lb)
	assert.Nil(lb.AddElement("1", "1"))
	assert.Equal(1, lb.Len())
}
